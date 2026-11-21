import SwiftUI

struct HomeView: View {
    @ObservedObject var core = Core.shared
    @State private var showFingerprint = false
    @State private var showConnect = false

    var body: some View {
        List {
            Section {
                VStack(spacing: 6) {
                    Image(systemName: "mountain.2.fill")
                        .font(.system(size: 44))
                        .foregroundColor(.accentColor)
                    Text("Tomorrow Drop").font(.headline)
                    Text(L.t("Transferred. Used. Done.", "Transferiu. Usou. Pronto.")).font(.subheadline).italic().foregroundColor(.secondary)
                    Text("\(L.t("This device", "Este dispositivo")): \(core.state.address)")
                        .font(.system(.footnote, design: .monospaced)).foregroundColor(.secondary)
                    Text(L.t("Found by other devices only while the app is open.",
                             "Encontrado por outros dispositivos somente com o app aberto."))
                        .font(.caption2).foregroundColor(.secondary).multilineTextAlignment(.center)
                    Button(L.t("View my fingerprint", "Ver minha impressão digital")) { showFingerprint = true }
                        .font(.footnote)
                }
                .frame(maxWidth: .infinity)
                .padding(.vertical, 8)
            }

            Section(header: Text(L.t("Devices found", "Dispositivos encontrados"))) {
                if core.state.devices.isEmpty {
                    Text(L.t("Looking for devices…", "Procurando dispositivos…")).foregroundColor(.secondary)
                    Text(L.t("Open Tomorrow Drop on the other device, on the same Wi-Fi network.",
                             "Abra o Tomorrow Drop no outro dispositivo, na mesma rede Wi-Fi."))
                        .font(.footnote).foregroundColor(.secondary)
                }
                ForEach(core.state.devices) { device in
                    DeviceRow(device: device)
                }
            }

            if !core.state.jobs.isEmpty {
                Section(
                    header: Text(L.t("In progress", "Em andamento")),
                    footer: Text(L.t("Keep the app open until these finish.", "Mantenha o app aberto até terminarem."))
                ) {
                    ForEach(core.state.jobs) { job in
                        JobRow(job: job)
                    }
                }
            }

            Section {
                Button(L.t("Connect by IP", "Conectar por IP")) { showConnect = true }
                NavigationLink(destination: SettingsView()) {
                    Label(L.t("Settings", "Configurações"), systemImage: "gearshape")
                }
            }

            if let notice = core.state.notice, !notice.isEmpty {
                Section { Text(notice).font(.footnote).foregroundColor(.orange) }
            }
        }
        .listStyle(.insetGrouped)
        .navigationTitle("Tomorrow Drop")
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .navigationBarTrailing) {
                Button { core.searchNow() } label: { Image(systemName: "arrow.clockwise") }
                    .accessibilityLabel(L.t("Refresh", "Atualizar"))
            }
        }
        .sheet(isPresented: $showFingerprint) { FingerprintView() }
        .sheet(isPresented: $showConnect) { ConnectView() }
    }
}

struct DeviceRow: View {
    let device: AppState.Device
    @ObservedObject var core = Core.shared

    var body: some View {
        if device.paired, let paired = core.state.paired.first(where: { $0.fingerprint == device.fingerprint }) {
            NavigationLink(destination: DeviceView(hex: paired.hex, addr: device.addr)) { label }
        } else {
            Button { core.pair(with: device.addr) } label: { label }
                .foregroundColor(.primary)
        }
    }

    private var label: some View {
        HStack {
            if device.paired {
                Image(systemName: "checkmark.circle.fill").foregroundColor(.green)
            } else {
                Image(systemName: "desktopcomputer").foregroundColor(.secondary)
            }
            VStack(alignment: .leading, spacing: 2) {
                Text(device.name)
                Text(device.addr).font(.system(.caption, design: .monospaced)).foregroundColor(.secondary)
            }
            Spacer()
            Text(device.paired ? L.t("Paired", "Pareado") : L.t("Tap to pair", "Toque para parear"))
                .font(.footnote).foregroundColor(device.paired ? .green : .secondary)
        }
    }
}

struct JobRow: View {
    let job: AppState.Job

    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(job.title).font(.subheadline)
            if job.state == "running" {
                ProgressView(value: job.fraction)
            } else {
                Text(job.message).font(.footnote)
                    .foregroundColor(job.state == "done" ? .green : .red)
            }
        }
    }
}

struct FingerprintView: View {
    @ObservedObject var core = Core.shared
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        NavigationView {
            VStack(alignment: .leading, spacing: 16) {
                Text(L.t("When pairing, compare the whole fingerprint, every character, with the one shown on the other device.",
                         "Ao parear, compare a impressão digital inteira, todos os caracteres, com a mostrada no outro dispositivo."))
                    .font(.footnote).foregroundColor(.secondary)
                ForEach(fingerprintLines(core.state.fingerprint), id: \.self) { line in
                    Text(line).font(.system(.title3, design: .monospaced))
                }
                CopyButton(text: core.state.fingerprint)
                Spacer()
            }
            .padding()
            .navigationTitle(L.t("This device's fingerprint", "Impressão digital deste dispositivo"))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .confirmationAction) {
                    Button(L.t("Close", "Fechar")) { dismiss() }
                }
            }
        }
    }
}

struct ConnectView: View {
    @ObservedObject var core = Core.shared
    @Environment(\.dismiss) private var dismiss
    @State private var address = ""

    var body: some View {
        NavigationView {
            Form {
                Section(footer: Text(L.t("Enter the IP address and port shown on the other device.",
                                         "Informe o endereço IP e a porta mostrados no outro dispositivo."))) {
                    TextField("192.168.1.42:47521", text: $address)
                        .keyboardType(.numbersAndPunctuation)
                        .autocapitalization(.none)
                }
                Button(L.t("Connect", "Conectar")) {
                    core.pair(with: address.trimmingCharacters(in: .whitespaces))
                    dismiss()
                }
                .disabled(address.trimmingCharacters(in: .whitespaces).isEmpty)
            }
            .navigationTitle(L.t("Connect by IP", "Conectar por IP"))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button(L.t("Cancel", "Cancelar")) { dismiss() }
                }
            }
        }
    }
}
