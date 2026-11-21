import SwiftUI

// PairSheet is the pairing: the fingerprints are compared on both devices, then
// the other device is named.
struct PairSheet: View {
    @ObservedObject var core = Core.shared
    @State private var name = ""

    var body: some View {
        NavigationView {
            if let pair = core.state.pair, pair.stage == "name" {
                Form {
                    Section(footer: Text(L.t("Name the other device so you can recognize it later.",
                                             "Dê um nome ao outro dispositivo para reconhecê-lo depois."))) {
                        TextField(pair.suggested, text: $name)
                    }
                    Button(L.t("Save", "Salvar")) { core.namePair(name.trimmingCharacters(in: .whitespaces)) }
                }
                .navigationTitle(L.t("Name the device", "Nomear o dispositivo"))
                .onAppear { name = pair.suggested }
            } else if let pair = core.state.pair {
                VStack(alignment: .leading, spacing: 16) {
                    Text(L.t("Compare these fingerprints with the other device. Confirm only if every character matches.",
                             "Compare estas impressões digitais com o outro dispositivo. Confirme só se cada caractere bater."))
                        .font(.footnote).foregroundColor(.secondary)
                    Text(L.t("This device", "Este dispositivo")).font(.headline)
                    ForEach(fingerprintLines(pair.local), id: \.self) { Text($0).font(.system(.body, design: .monospaced)) }
                    Text(L.t("Other device", "Outro dispositivo")).font(.headline)
                    ForEach(fingerprintLines(pair.remote), id: \.self) { Text($0).font(.system(.body, design: .monospaced)) }
                    Spacer()
                    HStack {
                        Button(L.t("Cancel", "Cancelar"), role: .cancel) { core.confirmPair(false) }
                            .buttonStyle(.bordered)
                        Button(L.t("Fingerprints match", "As impressões coincidem")) { core.confirmPair(true) }
                            .buttonStyle(.borderedProminent)
                    }
                    .frame(maxWidth: .infinity)
                }
                .padding()
                .navigationTitle(L.t("Pairing", "Pareamento"))
            }
        }
        .interactiveDismissDisabled(true)
    }
}

// OfferSheet is the decision for every offer, including from paired devices.
struct OfferSheet: View {
    @ObservedObject var core = Core.shared

    var body: some View {
        NavigationView {
            if let offer = core.state.offer {
                VStack(alignment: .leading, spacing: 14) {
                    Text(offer.text
                         ? L.t("\(offer.from) wants to send you a text:", "\(offer.from) quer enviar um texto para você:")
                         : L.t("\(offer.from) wants to send you:", "\(offer.from) quer enviar para você:"))
                        .font(.headline)
                    Text(L.t("\(offer.count) item(s) · \(ByteCountFormatter.string(fromByteCount: offer.bytes, countStyle: .file))",
                             "\(offer.count) item(ns) · \(ByteCountFormatter.string(fromByteCount: offer.bytes, countStyle: .file))"))
                        .foregroundColor(.secondary)
                    List(offer.names, id: \.self) { Text($0) }
                        .listStyle(.plain)
                    HStack {
                        Button(L.t("Decline", "Recusar"), role: .cancel) { core.answerOffer(false) }
                            .buttonStyle(.bordered)
                        Button(L.t("Accept", "Aceitar")) { core.answerOffer(true) }
                            .buttonStyle(.borderedProminent)
                    }
                    .frame(maxWidth: .infinity)
                }
                .padding()
                .navigationTitle(L.t("Receive files?", "Receber arquivos?"))
                .navigationBarTitleDisplayMode(.inline)
            }
        }
        .interactiveDismissDisabled(true)
    }
}
