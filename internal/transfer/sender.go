package transfer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"tomorrow-drop/internal/apperr"
	"tomorrow-drop/internal/protocol"
	"tomorrow-drop/internal/transport"
)

// SendSpec describes what to send. Files use Paths; text and links use Text.
type SendSpec struct {
	Kind  Kind
	Paths []string
	Text  string
}

// Progress reports how far a transfer is. Item values refer to the current
// item; Total values cover the whole offer.
type Progress struct {
	OfferID   string
	Item      int
	Name      string
	ItemSent  int64
	ItemSize  int64
	TotalSent int64
	TotalSize int64
}

// Result is what the recipient confirmed.
type Result struct {
	OfferID string
	Text    string
	Files   []string
}

type source struct {
	name string
	size int64
	r    io.Reader
	file *os.File
}

// Send offers the content on an authenticated session and streams it once the
// recipient accepts. The session must be in the trusted state. Cancelling ctx
// closes the session, which invalidates the transfer on both sides.
func Send(ctx context.Context, s *transport.Session, spec SendSpec, progress func(Progress)) (Result, error) {
	sources, items, err := prepare(spec)
	if err != nil {
		return Result{}, err
	}
	defer closeAll(sources)

	id := newOfferID()
	m := NewMachine()
	stop := context.AfterFunc(ctx, func() { _ = s.Close() })
	defer stop()

	if err := offer(ctx, s, m, id, spec.Kind, items); err != nil {
		return Result{}, err
	}
	_ = m.Move(StateTransferring)
	if err := stream(ctx, s, id, sources, progress); err != nil {
		_ = m.Move(StateFailed)
		return Result{}, err
	}
	_ = m.Move(StateFinishing)
	if err := finish(ctx, s, id); err != nil {
		_ = m.Move(StateFailed)
		return Result{}, err
	}
	_ = m.Move(StateDone)
	return Result{OfferID: id, Text: textOf(spec)}, nil
}

// offer sends the offer and waits for the recipient's answer.
func offer(ctx context.Context, s *transport.Session, m *Machine, id string, kind Kind, items []itemBody) error {
	if err := s.Send(protocol.MsgTransferOffer, offerBody{ID: id, Kind: kind, Items: items}); err != nil {
		return contextOr(ctx, err)
	}
	_ = m.Move(StateOffered)
	reply, err := s.ReceiveWithin(protocol.DecisionTimeout)
	if err != nil {
		return contextOr(ctx, err)
	}
	switch reply.Type {
	case protocol.MsgTransferAccept:
		_ = m.Move(StateAccepted)
		return nil
	case protocol.MsgTransferDecline:
		var d declineBody
		_ = reply.DecodeBody(&d)
		_ = m.Move(StateDeclined)
		return declineError(d.Reason)
	default:
		return fmt.Errorf("received %s instead of an answer to the offer: %w", reply.Type, apperr.ErrMalformed)
	}
}

// stream sends the data of every item, followed by its END message.
func stream(ctx context.Context, s *transport.Session, id string, sources []source, progress func(Progress)) error {
	var total int64
	for _, src := range sources {
		total += src.size
	}
	var totalSent int64
	for i, src := range sources {
		sent, err := streamItem(ctx, s, id, i, src, func(itemSent int64, n int) {
			totalSent += int64(n)
			if progress != nil {
				progress(Progress{OfferID: id, Item: i, Name: src.name, ItemSent: itemSent, ItemSize: src.size, TotalSent: totalSent, TotalSize: total})
			}
		})
		if err != nil {
			return err
		}
		end := endBody{ID: id, Index: i, Size: sent.size, SHA256: hex.EncodeToString(sent.sum)}
		if err := s.Send(protocol.MsgTransferEnd, end); err != nil {
			return contextOr(ctx, err)
		}
	}
	return nil
}

type itemSum struct {
	size int64
	sum  []byte
}

// streamItem reads one source in chunks, sends each chunk and hashes it.
func streamItem(ctx context.Context, s *transport.Session, id string, index int, src source, onChunk func(itemSent int64, n int)) (itemSum, error) {
	digest := sha256.New()
	var itemSent int64
	buf := make([]byte, chunkSize)
	for {
		n, rerr := src.r.Read(buf)
		if n > 0 {
			chunk := buf[:n]
			if err := s.Send(protocol.MsgTransferData, dataBody{ID: id, Index: index, Data: chunk}); err != nil {
				return itemSum{}, contextOr(ctx, err)
			}
			_, _ = digest.Write(chunk)
			itemSent += int64(n)
			onChunk(itemSent, n)
		}
		if errors.Is(rerr, io.EOF) {
			return itemSum{size: itemSent, sum: digest.Sum(nil)}, nil
		}
		if rerr != nil {
			return itemSum{}, fmt.Errorf("read %s: %w", src.name, rerr)
		}
	}
}

// finishTimeout is how long the sender waits for the recipient to publish.
var finishTimeout = protocol.PublishTimeout

// finish sends DONE and waits for the recipient to confirm publication.
func finish(ctx context.Context, s *transport.Session, id string) error {
	if err := s.Send(protocol.MsgTransferDone, idBody{ID: id}); err != nil {
		return contextOr(ctx, err)
	}
	res, err := s.ReceiveWithin(finishTimeout)
	if err != nil {
		return contextOr(ctx, err)
	}
	var r resultBody
	if err := res.DecodeBody(&r); err != nil || res.Type != protocol.MsgTransferResult {
		return fmt.Errorf("invalid transfer result: %w", apperr.ErrMalformed)
	}
	if !r.OK {
		return resultError(r.Reason)
	}
	return nil
}

// prepare opens every file before the offer is sent, so a missing or unreadable
// file fails the transfer before anything reaches the recipient.
func prepare(spec SendSpec) ([]source, []itemBody, error) {
	switch spec.Kind {
	case KindFile:
		if len(spec.Paths) == 0 {
			return nil, nil, fmt.Errorf("no files to send: %w", apperr.ErrMalformed)
		}
		var sources []source
		var items []itemBody
		for _, p := range spec.Paths {
			f, err := os.Open(p) //nolint:gosec // G304: the user explicitly chose this file to send
			if err != nil {
				closeAll(sources)
				return nil, nil, fmt.Errorf("open %s: %w", p, err)
			}
			info, err := f.Stat()
			if err != nil || !info.Mode().IsRegular() {
				_ = f.Close()
				closeAll(sources)
				return nil, nil, fmt.Errorf("%s is not a regular file: %w", p, apperr.ErrMalformed)
			}
			name := filepath.Base(p)
			if err := ValidateName(name); err != nil {
				_ = f.Close()
				closeAll(sources)
				return nil, nil, err
			}
			sources = append(sources, source{name: name, size: info.Size(), r: f, file: f})
			items = append(items, itemBody{Name: name, Size: info.Size()})
		}
		if err := ValidateOffer(Offer{ID: "0000000000000000", Kind: KindFile, Items: toItems(items)}); err != nil {
			closeAll(sources)
			return nil, nil, err
		}
		return sources, items, nil
	case KindText, KindLink:
		if spec.Kind == KindLink {
			if err := ValidateLink(spec.Text); err != nil {
				return nil, nil, err
			}
		}
		item := itemBody{Name: "", Size: int64(len(spec.Text))}
		if err := ValidateOffer(Offer{ID: "0000000000000000", Kind: spec.Kind, Items: toItems([]itemBody{item})}); err != nil {
			return nil, nil, err
		}
		src := source{size: item.Size, r: strings.NewReader(spec.Text)}
		return []source{src}, []itemBody{item}, nil
	default:
		return nil, nil, fmt.Errorf("send kind %q: %w", spec.Kind, apperr.ErrMalformed)
	}
}

func closeAll(sources []source) {
	for _, src := range sources {
		if src.file != nil {
			_ = src.file.Close()
		}
	}
}

func toItems(items []itemBody) []Item {
	out := make([]Item, 0, len(items))
	for _, it := range items {
		out = append(out, Item(it))
	}
	return out
}

func textOf(spec SendSpec) string {
	if spec.Kind == KindFile {
		return ""
	}
	return spec.Text
}

func declineError(reason string) error {
	switch reason {
	case reasonBusy:
		return apperr.ErrBusy
	case reasonInvalid:
		return apperr.ErrMalformed
	default:
		return apperr.ErrDeclined
	}
}

func resultError(reason string) error {
	switch reason {
	case reasonIntegrity:
		return apperr.ErrIntegrity
	case reasonSize:
		return apperr.ErrMalformed
	default:
		return apperr.ErrCanceled
	}
}

func contextOr(ctx context.Context, err error) error {
	if ctx.Err() != nil {
		return apperr.FromContext(ctx.Err())
	}
	return err
}
