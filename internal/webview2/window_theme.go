package webview2

func appThemeIsDark(highContrast bool, appsUseLightTheme uint64) bool {
	return !highContrast && appsUseLightTheme == 0
}
