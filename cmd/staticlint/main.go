package main

import (
	"strings"

	"github.com/Sayfargo/yax-url-shortener/cmd/staticlint/osexitanalyzer"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"golang.org/x/tools/go/analysis/passes/slog"
	"golang.org/x/tools/go/analysis/passes/tests"
	"golang.org/x/tools/go/analysis/passes/waitgroup"
	"honnef.co/go/tools/quickfix"
	"honnef.co/go/tools/simple"
	"honnef.co/go/tools/staticcheck"
	"honnef.co/go/tools/stylecheck"
)

// main формирует общий набор анализаторов и передаёт его в
// multichecker.Main для запуска проверки целевых пакетов.
func main() {
	mychecks := []*analysis.Analyzer{
		waitgroup.Analyzer,
		slog.Analyzer,
		tests.Analyzer,
		osexitanalyzer.NoOSExit,
	}
	analyzers := staticcheckAnalyzers()
	mychecks = append(mychecks, analyzers...)

	multichecker.Main(
		mychecks...,
	)

}

// staticcheckAnalyzers собирает и возвращает список анализаторов из набора
// honnef.co/go/tools: все проверки класса SA из пакета staticcheck,
// а также выборочные проверки.
func staticcheckAnalyzers() []*analysis.Analyzer {
	analyzers := make([]*analysis.Analyzer, 0, len(staticcheck.Analyzers))
	// Все анализаторы класса SA.
	for _, a := range staticcheck.Analyzers {
		if strings.HasPrefix(a.Analyzer.Name, "SA") {
			analyzers = append(analyzers, a.Analyzer)
		}
	}
	// Выборочные анализаторы для упрощения кода, соблюдения стиля и рефакторинга.
	extraChecks := map[string]bool{
		"S1000":  true,
		"S1028":  true,
		"ST1005": true,
		"ST1012": true,
		"QF1003": true,
	}

	for _, v := range simple.Analyzers {
		if extraChecks[v.Analyzer.Name] {
			analyzers = append(analyzers, v.Analyzer)
		}
	}

	for _, v := range stylecheck.Analyzers {
		if extraChecks[v.Analyzer.Name] {
			analyzers = append(analyzers, v.Analyzer)
		}
	}

	for _, v := range quickfix.Analyzers {
		if extraChecks[v.Analyzer.Name] {
			analyzers = append(analyzers, v.Analyzer)
		}
	}

	return analyzers
}
