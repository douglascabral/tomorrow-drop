import SwiftUI
import UIKit

@main
struct TomorrowDropApp: App {
    var body: some Scene {
        WindowGroup {
            RootView()
        }
    }
}

// L picks the Portuguese or English text by the device language.
enum L {
    static let isPT = (Locale.preferredLanguages.first ?? "").hasPrefix("pt")
    static func t(_ en: String, _ pt: String) -> String { isPT ? pt : en }
}

// chunks a grouped fingerprint (16 groups) into four lines of four groups.
func fingerprintLines(_ grouped: String) -> [String] {
    let groups = grouped.split(separator: " ").map(String.init)
    guard groups.count == 16 else { return [grouped] }
    return stride(from: 0, to: 16, by: 4).map { groups[$0..<$0 + 4].joined(separator: " ") }
}

struct RootView: View {
    @ObservedObject var core = Core.shared
    @Environment(\.scenePhase) private var phase

    var body: some View {
        NavigationView {
            HomeView()
        }
        .navigationViewStyle(.stack)
        .onAppear {
            core.start()
            setActive(phase == .active)
        }
        .onChange(of: phase) { p in setActive(p == .active) }
        .sheet(isPresented: Binding(get: { core.state.pair != nil }, set: { _ in })) {
            PairSheet()
        }
        .sheet(isPresented: Binding(get: { core.state.offer != nil }, set: { _ in })) {
            OfferSheet()
        }
        .sheet(isPresented: Binding(get: { core.state.received != nil }, set: { _ in })) {
            ReceivedTextView(text: core.state.received ?? "")
        }
        .alert(core.state.result?.title ?? "", isPresented: Binding(
            get: { core.state.result != nil && core.state.received == nil },
            set: { _ in }
        )) {
            Button("OK") { core.dismissResult() }
        } message: {
            Text(core.state.result?.message ?? "")
        }
    }

    // The app listens and searches only while it is on the screen (PRD §17); the
    // screen stays on while it is open.
    private func setActive(_ active: Bool) {
        core.setAvailable(active)
        UIApplication.shared.isIdleTimerDisabled = active
    }
}
