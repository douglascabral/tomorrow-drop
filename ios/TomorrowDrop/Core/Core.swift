import Foundation
import UIKit

// The state the core reports (see cmd/tomorrow-drop-ios/state.go).
struct AppState: Codable {
    var fingerprint = ""
    var address = ""
    var listening = false
    var devices: [Device] = []
    var paired: [Paired] = []
    var pair: PairQuestion?
    var offer: OfferQuestion?
    var jobs: [Job] = []          // operations in progress only
    var notice: String?
    var result: Result?           // the outcome of the last operation, shown once
    var received: String?         // a text received, shown until closed

    struct Result: Codable {
        var ok: Bool
        var title: String
        var message: String
    }

    struct Device: Codable, Identifiable {
        var id: String { addr }
        var name: String
        var addr: String
        var paired: Bool
        var fingerprint: String?
    }

    struct Paired: Codable, Identifiable {
        var id: String { hex }
        var name: String
        var fingerprint: String
        var hex: String
        var addr: String?
        // "online", "offline" or "unknown": the last reachability check (presence.go).
        var presence: String?
    }

    struct PairQuestion: Codable {
        var local: String
        var remote: String
        var stage: String
        var suggested: String
        var incoming: Bool
    }

    struct OfferQuestion: Codable {
        var from: String
        var count: Int
        var bytes: Int64
        var names: [String]
        var text: Bool
    }

    struct Job: Codable, Identifiable {
        var id: Int
        var title: String
        var fraction: Double
        var state: String
        var message: String
    }
}

// takeString copies a string the core allocated and frees it.
private func takeString(_ p: UnsafeMutablePointer<CChar>?) -> String {
    guard let p = p else { return "" }
    let s = String(cString: p)
    tdFree(p)
    return s
}

// withC passes a Swift string to the core as a C string for the duration of the call.
private func withC<R>(_ s: String, _ body: (UnsafeMutablePointer<CChar>) -> R) -> R {
    var bytes = Array(s.utf8CString)
    return bytes.withUnsafeMutableBufferPointer { body($0.baseAddress!) }
}

// Core is the application state the interface shows. It asks the Go core for
// the state every second and sends the person's answers back.
final class Core: ObservableObject {
    static let shared = Core()

    @Published private(set) var state = AppState()
    private var timer: Timer?
    private var started = false

    func start() {
        guard !started else { return }
        started = true
        let fm = FileManager.default
        let support = try! fm.url(for: .applicationSupportDirectory, in: .userDomainMask, appropriateFor: nil, create: true)
            .appendingPathComponent("tomorrow-drop", isDirectory: true)
        // Received files go to Documents, so the Files app shows them (see Info.plist).
        let docs = try! fm.url(for: .documentDirectory, in: .userDomainMask, appropriateFor: nil, create: true)
        let name = "\(UIDevice.current.model) · \(UIDevice.current.systemName) \(UIDevice.current.systemVersion)"
        let lang = (Locale.preferredLanguages.first ?? "").hasPrefix("pt") ? "pt-BR" : "en"
        let error = withC(support.path) { h in
            withC(docs.path) { d in
                withC(name) { n in
                    withC(lang) { l in takeString(tdStart(h, d, n, l)) }
                }
            }
        }
        if !error.isEmpty {
            state.notice = error
        }
        timer = Timer.scheduledTimer(withTimeInterval: 1, repeats: true) { [weak self] _ in self?.refresh() }
        refresh()
    }

    func refresh() {
        let json = takeString(tdState())
        guard let data = json.data(using: .utf8) else { return }
        do {
            var s = try JSONDecoder().decode(AppState.self, from: data)
            state = s
        } catch {
            // Keep the last good state, but say what failed instead of showing nothing.
            var s = state
            s.notice = "Estado inválido do núcleo: \(error)"
            state = s
        }
    }

    // searchNow asks the network for devices at once.
    func searchNow() {
        tdSearch()
        refresh()
    }

    // dismissResult and dismissReceived close what the person has read. Nothing
    // of them is kept: the app has no history.
    func dismissResult() {
        tdDismissResult()
        refresh()
    }

    func dismissReceived() {
        tdDismissReceived()
        refresh()
    }

    func setAvailable(_ on: Bool) {
        tdSetAvailable(on ? 1 : 0)
        refresh()
    }

    func pair(with addr: String) {
        withC(addr) { tdPairStart($0) }
        refresh()
    }

    func confirmPair(_ yes: Bool) {
        tdPairConfirm(yes ? 1 : 0)
        refresh()
    }

    func namePair(_ name: String) {
        withC(name) { tdPairName($0) }
        refresh()
    }

    func answerOffer(_ yes: Bool) {
        tdOfferAnswer(yes ? 1 : 0)
        refresh()
    }

    func sendFiles(_ paths: [String], to addr: String) {
        guard let data = try? JSONSerialization.data(withJSONObject: paths), let json = String(data: data, encoding: .utf8) else { return }
        _ = withC(addr) { a in withC(json) { tdSendFiles(a, $0) } }
        refresh()
    }

    func sendText(_ text: String, to addr: String) {
        _ = withC(addr) { a in withC(text) { tdSendText(a, $0) } }
        refresh()
    }

    func rename(_ hex: String, to name: String) {
        _ = withC(hex) { h in withC(name) { tdRename(h, $0) } }
        refresh()
    }

    func unpair(_ hex: String) {
        _ = withC(hex) { tdUnpair($0) }
        refresh()
    }
}
