// Package osexitanalyzer анализатор для поиска и запрета прямых вызовов os.Exit()
// в пакете main в функции main().
package osexitanalyzer

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

var NoOSExist = &analysis.Analyzer{
	Name: "noosexist",
	Doc:  "forbids direct calls to os.Exit in main package",
	Run:  run,
}

func run(pass *analysis.Pass) (any, error) {
	// Сначала определим что текущий пакет действительно main
	if pass.Pkg.Name() != "main" {
		return nil, nil
	}

	for _, file := range pass.Files {

		// Определим именно декларацию main функции
		for _, decl := range file.Decls {
			fnDecl, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			if fnDecl.Recv != nil || fnDecl.Name.Name != "main" || fnDecl.Body == nil {
				continue
			}
			// Проходим по телу функции main и ищем любой вызов функции
			ast.Inspect(fnDecl.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				// Нужен именно вызов функции через точку (селектор), так как os.Exit
				// это вызов из стандартной библиотеки, нужна левая и правая часть от точки
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				// Ищем по ключу в мапе используемых в коде объектов значение самой сущности
				obj, exist := pass.TypesInfo.Uses[sel.Sel]
				if !exist {
					return true
				}
				// Определим через type assertion в тип функцию, чтобы получить имя пакета и имя функции
				fn, isFunc := obj.(*types.Func)
				if !isFunc || fn.Pkg() == nil {
					return true
				}
				// Если это вызов os.Exit, то репортим
				if fn.Pkg().Path() == "os" && fn.Name() == "Exit" {
					pass.Reportf(call.Pos(), "direct call to os.Exit is forbidden")
				}

				return true
			})

		}
	}

	return nil, nil
}
