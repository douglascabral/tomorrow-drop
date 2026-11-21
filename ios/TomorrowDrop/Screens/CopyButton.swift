import SwiftUI
import UIKit

// CopyButton copies a text to the clipboard and confirms it on the button itself
// for a moment: the label changes to "Copied" with a check mark, then returns.
struct CopyButton: View {
    let text: String
    var prominent = false

    @State private var copied = false

    var body: some View {
        Button {
            UIPasteboard.general.string = text
            UINotificationFeedbackGenerator().notificationOccurred(.success)
            withAnimation { copied = true }
            DispatchQueue.main.asyncAfter(deadline: .now() + 2) {
                withAnimation { copied = false }
            }
        } label: {
            Label(
                copied ? L.t("Copied", "Copiado") : L.t("Copy", "Copiar"),
                systemImage: copied ? "checkmark.circle.fill" : "doc.on.doc"
            )
            .frame(maxWidth: prominent ? .infinity : nil)
        }
        .foregroundColor(copied ? .green : nil)
        .modifier(CopyStyle(prominent: prominent))
        .accessibilityLabel(copied ? L.t("Copied", "Copiado") : L.t("Copy", "Copiar"))
    }
}

private struct CopyStyle: ViewModifier {
    let prominent: Bool

    func body(content: Content) -> some View {
        if prominent {
            content.buttonStyle(.borderedProminent)
        } else {
            content
        }
    }
}
