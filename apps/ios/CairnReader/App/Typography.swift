import SwiftUI
import UIKit

/// The bundled brand faces (PostScript names): the same files the React Native app loads
/// (`FontFamily` in `apps/mobile/src/constants/theme.ts`), registered through `UIAppFonts`.
enum BrandFont: String, CaseIterable {
    case inter = "Inter-Regular"
    case interMedium = "Inter-Medium"
    case interSemiBold = "Inter-SemiBold"
    case interBold = "Inter-Bold"
    case crimsonPro = "CrimsonPro-Regular"
    case crimsonProMedium = "CrimsonPro-Medium"
    case crimsonProSemiBold = "CrimsonPro-SemiBold"
    case crimsonProBold = "CrimsonPro-Bold"
    case atkinson = "AtkinsonHyperlegibleNext-Regular"
    case atkinsonItalic = "AtkinsonHyperlegibleNext-Italic"
    case atkinsonBold = "AtkinsonHyperlegibleNext-Bold"

    /// A SwiftUI font that scales with Dynamic Type like `style`.
    func font(size: CGFloat, relativeTo style: Font.TextStyle) -> Font {
        .custom(rawValue, size: size, relativeTo: style)
    }

    /// A UIKit font that scales with Dynamic Type like `style`, for the bar appearances.
    func uiFont(size: CGFloat, relativeTo style: UIFont.TextStyle) -> UIFont {
        // Every case is bundled; CairnReaderTests fails if one does not load.
        UIFontMetrics(forTextStyle: style).scaledFont(for: UIFont(name: rawValue, size: size)!)
    }
}

/// One font per text role. Inter is the UI font, Crimson Pro the heading font and Atkinson
/// Hyperlegible Next the article body font. Sizes follow `FontSizes` in `theme.ts`.
enum Typography {
    static let largeTitle = BrandFont.crimsonProBold.font(size: 32, relativeTo: .largeTitle)
    static let title = BrandFont.crimsonProSemiBold.font(size: 24, relativeTo: .title2)
    static let heading = BrandFont.crimsonPro.font(size: 18, relativeTo: .title3)

    /// Applied once at the root, so views inherit it without a font modifier.
    static let body = BrandFont.inter.font(size: 16, relativeTo: .body)
    static let bodyMedium = BrandFont.interMedium.font(size: 16, relativeTo: .body)
    static let bodySemiBold = BrandFont.interSemiBold.font(size: 16, relativeTo: .body)
    static let bodyBold = BrandFont.interBold.font(size: 16, relativeTo: .body)
    static let subheadline = BrandFont.inter.font(size: 14, relativeTo: .subheadline)
    static let caption = BrandFont.inter.font(size: 12, relativeTo: .caption)

    static let content = BrandFont.atkinson.font(size: 16, relativeTo: .body)
    static let contentItalic = BrandFont.atkinsonItalic.font(size: 16, relativeTo: .body)
    static let contentBold = BrandFont.atkinsonBold.font(size: 16, relativeTo: .body)

    /// Navigation bar titles and tab bar items, which SwiftUI's `.font` does not reach.
    static func applyBarAppearance() {
        let navigationBar = UINavigationBar.appearance()
        navigationBar.titleTextAttributes = [.font: BrandFont.interSemiBold.uiFont(size: 17, relativeTo: .headline)]
        navigationBar.largeTitleTextAttributes = [.font: BrandFont.crimsonProBold.uiFont(size: 34, relativeTo: .largeTitle)]
        UITabBarItem.appearance().setTitleTextAttributes(
            [.font: BrandFont.interMedium.uiFont(size: 10, relativeTo: .caption2)], for: .normal)
    }
}
