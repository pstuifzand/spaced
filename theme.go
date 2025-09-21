package main

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

type SpacedRepetitionTheme struct{}

var _ fyne.Theme = (*SpacedRepetitionTheme)(nil)

func (t *SpacedRepetitionTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		if variant == theme.VariantLight {
			return color.NRGBA{248, 249, 250, 255} // Very light gray
		}
		return color.NRGBA{18, 18, 18, 255} // Dark background

	case theme.ColorNameButton:
		if variant == theme.VariantLight {
			return color.NRGBA{255, 255, 255, 255} // White buttons
		}
		return color.NRGBA{44, 44, 46, 255} // Dark gray buttons

	case theme.ColorNameForeground:
		if variant == theme.VariantLight {
			return color.NRGBA{33, 37, 41, 255} // Dark text
		}
		return color.NRGBA{255, 255, 255, 255} // Light text

	case theme.ColorNamePrimary:
		return color.NRGBA{52, 144, 220, 255} // Nice blue

	case theme.ColorNameSuccess:
		return color.NRGBA{40, 167, 69, 255} // Green for "Good"

	case theme.ColorNameError:
		return color.NRGBA{220, 53, 69, 255} // Red for "Again"

	case theme.ColorNameWarning:
		return color.NRGBA{255, 193, 7, 255} // Yellow/Orange for "Hard"

	case theme.ColorNameHover:
		if variant == theme.VariantLight {
			return color.NRGBA{248, 249, 250, 255}
		}
		return color.NRGBA{66, 66, 66, 255}

	case theme.ColorNamePressed:
		if variant == theme.VariantLight {
			return color.NRGBA{233, 236, 239, 255}
		}
		return color.NRGBA{88, 88, 88, 255}

	case theme.ColorNameShadow:
		return color.NRGBA{0, 0, 0, 20} // Subtle shadow

	default:
		return theme.DefaultTheme().Color(name, variant)
	}
}

func (t *SpacedRepetitionTheme) Font(style fyne.TextStyle) fyne.Resource {
	if style.Monospace {
		return theme.DefaultTheme().Font(style)
	}
	return theme.DefaultTheme().Font(style)
}

func (t *SpacedRepetitionTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (t *SpacedRepetitionTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNameText:
		return 16 // Base text size

	case theme.SizeNameHeadingText:
		return 24 // Heading text - larger for questions

	case theme.SizeNameSubHeadingText:
		return 20 // Sub-heading text - for answers

	case theme.SizeNameCaptionText:
		return 12 // Small text for stats

	case theme.SizeNamePadding:
		return 8 // More generous padding

	case theme.SizeNameInnerPadding:
		return 12 // Internal padding

	case theme.SizeNameScrollBar:
		return 12

	case theme.SizeNameInlineIcon:
		return 20

	case theme.SizeNameInputBorder:
		return 2

	case theme.SizeNameInputRadius:
		return 8 // Rounded corners

	default:
		return theme.DefaultTheme().Size(name)
	}
}

// Custom colors for rating buttons
var (
	AgainColor = color.NRGBA{220, 53, 69, 255}   // Red
	HardColor  = color.NRGBA{255, 133, 27, 255}  // Orange
	GoodColor  = color.NRGBA{40, 167, 69, 255}   // Green
	EasyColor  = color.NRGBA{52, 144, 220, 255}  // Blue
)