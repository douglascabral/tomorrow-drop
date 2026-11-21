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
                    Text(L.t("Visible only while the app is open.",
                             "Visível só com o app aberto."))
                        .font(.caption2).foregroundColor(.secondary).multilineTextAlignment(.center)
                    Button(L.t("View my fingerprint", "Ver minha impressão digital")) { showFingerprint = true }
                        .font(.footnote)
                }
                .frame(maxWidth: .infinity)
                .padding(.vertical, 8)
            }

            // Paired devices come from the trust store, so they stay here even when
            // no announcement reaches this device (another VLAN, or the app was reopened).
            Section(header: Text(L.t("Paired devices", "Dispositivos pareados"))) {
                if core.state.paired.isEmpty {
                    Text(L.t("No paired devices yet.", "Nenhum dispositivo pareado ainda.")).foregroundColor(.secondary)
                }
                ForEach(core.state.paired) { p in
                    PairedRow(paired: p)
                }
            }

            Section(header: Text(L.t("Devices found", "Dispositivos encontrados"))) {
                let found = core.state.devices.filter { !$0.paired }
                if found.isEmpty {
                    Text(L.t("Looking for devices…", "Procurando dispositivos…")).foregroundColor(.secondary)
                    Text(L.t("Open Tomorrow Drop on the other device, on the same Wi-Fi network.",
                             "Abra o Tomorrow Drop no outro dispositivo, na mesma rede Wi-Fi."))
                        .font(.footnote).foregroundColor(.secondary)
                }
                ForEach(found) { device in
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

// PairedRow is a paired device on the home screen. Its actions (send, receive)
// are on the device screen. A device whose address is not known yet cannot be
// opened from here until it is connected to by IP once. Only two states are
// shown: Online (green) when it answered its last check, Offline (red) otherwise.
struct PairedRow: View {
    let paired: AppState.Paired

    var body: some View {
        if let addr = paired.addr, !addr.isEmpty {
            NavigationLink(destination: DeviceView(hex: paired.hex, addr: addr)) { label }
        } else {
            label
        }
    }

    private var online: Bool { paired.presence == "online" }

    private var label: some View {
        DeviceLine(
            icon: online ? "checkmark.circle.fill" : "xmark.circle.fill",
            iconColor: online ? .green : .red,
            title: paired.name,
            subtitle: paired.addr ?? L.t("No known address", "Sem endereço conhecido"),
            status: online ? L.t("Online", "Online") : L.t("Offline", "Offline"),
            statusColor: online ? .green : .red
        )
    }
}

// DeviceLine is the one layout of a device line, for paired and found devices:
// icon, name and address, then the status. The icon column has a fixed width and
// the status does not wrap, so the lines line up.
struct DeviceLine: View {
    let icon: String
    let iconColor: Color
    let title: String
    let subtitle: String
    let status: String
    let statusColor: Color

    var body: some View {
        HStack(spacing: 12) {
            Image(systemName: icon)
                .foregroundColor(iconColor)
                .frame(width: 24)
            VStack(alignment: .leading, spacing: 2) {
                Text(title).lineLimit(1)
                Text(subtitle)
                    .font(.system(.caption, design: .monospaced)).foregroundColor(.secondary)
                    .lineLimit(1)
            }
            Spacer(minLength: 8)
            Text(status)
                .font(.footnote).foregroundColor(statusColor)
                .fixedSize()
        }
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

    // A found device that is not paired yet. Its status is not a reachability
    // state, so it keeps its own neutral text.
    private var label: some View {
        DeviceLine(
            icon: "desktopcomputer",
            iconColor: .secondary,
            title: device.name,
            subtitle: device.addr,
            status: L.t("Tap to pair", "Toque para parear"),
            statusColor: .secondary
        )
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
    @State private var ip = ""
    // The port the other device listens on; it is the usual one, so it is filled in.
    @State private var port = "47521"

    private var ready: Bool {
        !ip.trimmingCharacters(in: .whitespaces).isEmpty && !port.trimmingCharacters(in: .whitespaces).isEmpty
    }

    var body: some View {
        NavigationView {
            Form {
                Section(
                    header: Text(L.t("IP address", "Endereço IP")),
                    footer: Text(L.t("Enter the IP address and port shown on the other device.",
                                     "Informe o endereço IP e a porta mostrados no outro dispositivo."))
                ) {
                    TextField("192.168.1.42", text: $ip)
                        .keyboardType(.numbersAndPunctuation)
                        .autocapitalization(.none)
                        .disableAutocorrection(true)
                }
                Section(header: Text(L.t("Port", "Porta"))) {
                    TextField("47521", text: $port)
                        .keyboardType(.numberPad)
                }
                Button(L.t("Connect", "Conectar")) {
                    let address = ip.trimmingCharacters(in: .whitespaces) + ":" + port.trimmingCharacters(in: .whitespaces)
                    core.pair(with: address)
                    dismiss()
                }
                .disabled(!ready)
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
