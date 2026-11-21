# Performance and stability

Task 066. Measured here, on loopback; not on a real network.

## Throughput

`BenchmarkTransferLargeFile` sends a 64 MiB file over an authenticated TLS 1.3 connection on loopback, including pairing, the offer, the decision, integrity checks and publishing.

```sh
go test -run='^$' -bench=BenchmarkTransferLargeFile -benchtime=3x ./internal/transfer/
```

Result on Apple M4, 16 GiB RAM, macOS: 249.6 ms per 64 MiB, about **269 MB/s**. This measures the protocol and the storage path. A real Wi-Fi or Ethernet link is slower and was not measured.

## Stability

`TestManySequentialTransfersStayStable` sends 20 files of 256 KiB over new connections between the same paired devices. Every file must arrive intact and no staging file may remain. It passes.

## Not measured

- Slow or lossy links, network failures in the middle of large transfers on a real network (see the cases of task 024).
- Many devices in one VLAN (the discovery list is capped at 256 entries and tested for that cap, but not on a network with that many devices).
- Memory use under load. Transfers stream in 32 KiB chunks, so memory does not grow with file size, but this was not profiled.
