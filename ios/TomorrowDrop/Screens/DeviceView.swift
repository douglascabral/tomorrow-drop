import SwiftUI
import UniformTypeIdentifiers

struct DeviceView: View {
    let hex: String
    let addr: String
    @ObservedObject var core = Core.shared

    @State private var showFiles = false
    @State private var showText = false
    @State private var showRename = false
    @State private var showUnpair = false
    @State private var newName = ""
    @State private var message: String?

    private var paired: AppState.Paired? { core.state.paired.first { $0.hex == hex } }

    var body: some View {
        List {
            Section {
                VStack(spacing: 6) {
                    Image(systemName: "desktopcomputer").font(.system(size: 40)).foregroundColor(.accentColor)
                    Text(paired?.name ?? addr).font(.headline)
                    Text(addr).font(.system(.footnote, design: .monospaced)).foregroundColor(.secondary)
                    Label(L.t("Paired", "Pareado"), systemImage: "checkmark.circle.fill")
                        .font(.footnote).foregroundColor(.green)
                }
                .frame(maxWidth: .infinity)
            }

            Section {
                Button { showFiles = true } label: { Label(L.t("Send files", "Enviar arquivos"), systemImage: "square.and.arrow.up") }
                Button { showText = true } label: { Label(L.t("Send text", "Enviar texto"), systemImage: "doc.text") }
                Button { sendClipboard() } label: { Label(L.t("Send clipboard", "Enviar área de transferência"), systemImage: "doc.on.clipboard") }
            }

            Section(header: Text(L.t("Fingerprint (SHA-256)", "Impressão digital (SHA-256)"))) {
                ForEach(fingerprintLines(paired?.fingerprint ?? ""), id: \.self) { line in
                    Text(line).font(.system(.footnote, design: .monospaced))
                }
            }

            if !core.state.jobs.isEmpty {
                Section(
                    header: Text(L.t("In progress", "Em andamento")),
                    footer: Text(L.t("Keep the app open until these finish.", "Mantenha o app aberto até terminarem."))
                ) {
                    ForEach(core.state.jobs) { job in JobRow(job: job) }
                }
            }

            Section {
                Button(L.t("Rename", "Renomear")) {
                    newName = paired?.name ?? ""
                    showRename = true
                }
                Button(L.t("Unpair", "Desparear"), role: .destructive) { showUnpair = true }
            }

            if let message = message {
                Section { Text(message).font(.footnote).foregroundColor(.orange) }
            }
        }
        .listStyle(.insetGrouped)
        .navigationTitle(paired?.name ?? addr)
        .navigationBarTitleDisplayMode(.inline)
        .fileImporter(isPresented: $showFiles, allowedContentTypes: [.item], allowsMultipleSelection: true) { result in
            if case .success(let urls) = result { sendFiles(urls) }
        }
        .sheet(isPresented: $showText) { TextSendView(addr: addr) }
        .alert(L.t("Rename device", "Renomear dispositivo"), isPresented: $showRename) {
            TextField(L.t("Name", "Nome"), text: $newName)
            Button(L.t("Save", "Salvar")) {
                let name = newName.trimmingCharacters(in: .whitespaces)
                if !name.isEmpty { core.rename(hex, to: name) }
            }
            Button(L.t("Cancel", "Cancelar"), role: .cancel) {}
        }
        .confirmationDialog(
            L.t("Stop trusting this device? You will need to pair again to exchange files.",
                "Desparear este dispositivo? Você precisará parear de novo para trocar arquivos."),
            isPresented: $showUnpair, titleVisibility: .visible
        ) {
            Button(L.t("Unpair", "Desparear"), role: .destructive) { core.unpair(hex) }
            Button(L.t("Cancel", "Cancelar"), role: .cancel) {}
        }
    }

    // sendFiles copies the chosen files into the app before sending: the picked
    // URLs are only readable while they are open.
    private func sendFiles(_ urls: [URL]) {
        var paths: [String] = []
        let dir = FileManager.default.temporaryDirectory.appendingPathComponent("send-\(UUID().uuidString)", isDirectory: true)
        try? FileManager.default.createDirectory(at: dir, withIntermediateDirectories: true)
        for url in urls {
            let scoped = url.startAccessingSecurityScopedResource()
            defer { if scoped { url.stopAccessingSecurityScopedResource() } }
            let target = dir.appendingPathComponent(url.lastPathComponent)
            if (try? FileManager.default.copyItem(at: url, to: target)) != nil {
                paths.append(target.path)
            }
        }
        if !paths.isEmpty { core.sendFiles(paths, to: addr) }
    }

    // sendClipboard sends a picture or a text from the clipboard.
    private func sendClipboard() {
        let board = UIPasteboard.general
        if let image = board.image, let png = image.pngData() {
            let dir = FileManager.default.temporaryDirectory.appendingPathComponent("clip-\(UUID().uuidString)", isDirectory: true)
            try? FileManager.default.createDirectory(at: dir, withIntermediateDirectories: true)
            let stamp = ISO8601DateFormatter().string(from: Date()).replacingOccurrences(of: ":", with: "-")
            let file = dir.appendingPathComponent("Captura-\(stamp).png")
            if (try? png.write(to: file)) != nil {
                core.sendFiles([file.path], to: addr)
                return
            }
        }
        if let text = board.string, !text.isEmpty {
            core.sendText(text, to: addr)
            return
        }
        message = L.t("The clipboard is empty.", "A área de transferência está vazia.")
    }
}

struct TextSendView: View {
    let addr: String
    @ObservedObject var core = Core.shared
    @Environment(\.dismiss) private var dismiss
    @State private var text = ""

    var body: some View {
        NavigationView {
            Form {
                TextEditor(text: $text).frame(minHeight: 160)
            }
            .navigationTitle(L.t("Send text", "Enviar texto"))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button(L.t("Cancel", "Cancelar")) { dismiss() }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button(L.t("Send", "Enviar")) {
                        core.sendText(text, to: addr)
                        dismiss()
                    }
                    .disabled(text.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
                }
            }
        }
    }
}

struct SettingsView: View {
    @ObservedObject var core = Core.shared

    var body: some View {
        List {
            Section(header: Text(L.t("Paired devices", "Dispositivos pareados"))) {
                if core.state.paired.isEmpty {
                    Text(L.t("No paired devices yet.", "Nenhum dispositivo pareado ainda.")).foregroundColor(.secondary)
                }
                ForEach(core.state.paired) { p in
                    NavigationLink(destination: DeviceView(hex: p.hex, addr: p.addr ?? "")) {
                        VStack(alignment: .leading, spacing: 2) {
                            Text(p.name)
                            Text(p.fingerprint).font(.system(.caption, design: .monospaced)).foregroundColor(.secondary)
                        }
                    }
                }
            }
            Section {
                Text(L.t("Tomorrow Drop: transfers on the local network, with no accounts or cloud.",
                         "Tomorrow Drop: transferências na rede local, sem contas nem nuvem."))
                    .font(.footnote).foregroundColor(.secondary)
            }
        }
        .listStyle(.insetGrouped)
        .navigationTitle(L.t("Settings", "Configurações"))
        .navigationBarTitleDisplayMode(.inline)
    }
}

// ReceivedTextView shows a text received from another device, so it can be read
// and copied at once, as on the other systems.
struct ReceivedTextView: View {
    let text: String
    @ObservedObject var core = Core.shared

    var body: some View {
        NavigationView {
            VStack(alignment: .leading, spacing: 12) {
                ScrollView {
                    Text(text)
                        .frame(maxWidth: .infinity, alignment: .leading)
                        .textSelection(.enabled)
                }
                CopyButton(text: text, prominent: true)
            }
            .padding()
            .navigationTitle(L.t("Received text", "Texto recebido"))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .confirmationAction) {
                    Button(L.t("Close", "Fechar")) { core.dismissReceived() }
                }
            }
        }
        .interactiveDismissDisabled(true)
    }
}
