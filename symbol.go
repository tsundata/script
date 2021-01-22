package script

import (
	"fmt"
	"log"
	"strings"

	"github.com/tsundata/script/utils/collection"
)

type Symbol interface{}

type VarSymbol struct {
	Name       string
	Type       Symbol
	ScopeLevel int
}

func NewVarSymbol(name string, t Symbol) *VarSymbol {
	return &VarSymbol{Name: name, Type: t, ScopeLevel: 0}
}

func (s *VarSymbol) String() string {
	return fmt.Sprintf("<VarSymbol(name=%s, type=%v)>", s.Name, s.Type)
}

type BuiltinTypeSymbol struct {
	Name       string
	Type       Symbol
	ScopeLevel int
}

func NewBuiltinTypeSymbol(name string) *BuiltinTypeSymbol {
	return &BuiltinTypeSymbol{Name: name, ScopeLevel: 0}
}

func (s *BuiltinTypeSymbol) String() string {
	return fmt.Sprintf("<BuiltinTypeSymbol(name=%s)>", s.Name)
}

type FunctionSymbol struct {
	Package      string
	Name         string
	FormalParams []Ast
	ReturnType   Ast
	BlockAst     Ast
	ScopeLevel   int
	Call         CallFunc
}

func NewFunctionSymbol(name string) *FunctionSymbol {
	return &FunctionSymbol{Name: name, ScopeLevel: 0}
}

func (s *FunctionSymbol) String() string {
	return fmt.Sprintf("<FunctionSymbol(name=%s, package=%v, parameters=%v, return=%v)>", s.Name, s.Package, s.FormalParams, s.ReturnType)
}

type ScopedSymbolTable struct {
	symbols        *collection.OrderedDict
	ScopeName      string
	ScopeLevel     int
	EnclosingScope *ScopedSymbolTable
}

func NewScopedSymbolTable(scopeName string, scopeLevel int, enclosingScope *ScopedSymbolTable) *ScopedSymbolTable {
	table := &ScopedSymbolTable{
		symbols:        collection.NewOrderedDict(),
		ScopeName:      scopeName,
		ScopeLevel:     scopeLevel,
		EnclosingScope: enclosingScope,
	}
	table.Insert(NewBuiltinTypeSymbol("INTEGER"))
	table.Insert(NewBuiltinTypeSymbol("FLOAT"))
	table.Insert(NewBuiltinTypeSymbol("BOOL"))
	table.Insert(NewBuiltinTypeSymbol("STRING"))
	table.Insert(NewBuiltinTypeSymbol("LIST"))
	table.Insert(NewBuiltinTypeSymbol("DICT"))
	return table
}

func (t *ScopedSymbolTable) String() string {
	if t == nil {
		return ""
	}

	var lines []string

	lines = append(lines, fmt.Sprintf("Scope name : %s", t.ScopeName))
	lines = append(lines, fmt.Sprintf("Scope level : %d", t.ScopeLevel))

	if t.EnclosingScope != nil {
		lines = append(lines, fmt.Sprintf("Enclosing scope : %s", t.EnclosingScope.ScopeName))
	}

	lines = append(lines, "------------------------------------")
	lines = append(lines, "Scope (Scoped symbol table) contents")

	i := 0
	for v := range t.symbols.Iterate() {
		i++
		lines = append(lines, fmt.Sprintf("%6d: %v", i, v))
	}

	return fmt.Sprintf("\nSCOPE (SCOPED SYMBOL TABLE)\n===========================\n%s\n", strings.Join(lines, "\n"))
}

func (t *ScopedSymbolTable) Insert(symbol Symbol) {
	log.Printf("Insert: %s\n", symbol)
	var name string
	if s, ok := symbol.(*VarSymbol); ok {
		name = s.Name
		s.ScopeLevel = t.ScopeLevel
		t.symbols.Set(name, s)
		return
	}
	if s, ok := symbol.(*BuiltinTypeSymbol); ok {
		name = s.Name
		s.ScopeLevel = t.ScopeLevel
		t.symbols.Set(name, s)
		return
	}
	if s, ok := symbol.(*FunctionSymbol); ok {
		name = s.Name
		s.ScopeLevel = t.ScopeLevel
		if s.Package != "" {
			t.symbols.Set(fmt.Sprintf("%s.%s", s.Package, s.Name), s)
		} else {
			t.symbols.Set(name, s)
		}
		return
	}
}

func (t *ScopedSymbolTable) Lookup(name string, currentScopeOnly bool) Symbol {
	log.Printf("Lookup: %s. (Scope name: %s)\n", name, t.ScopeName)
	s := t.symbols.Get(name)
	if s != nil {
		return s.(Symbol)
	}
	if currentScopeOnly {
		return nil
	}

	if t.EnclosingScope != nil {
		return t.EnclosingScope.Lookup(name, false)
	}
	return nil
}

type SemanticAnalyzer struct {
	CurrentScope *ScopedSymbolTable
}

func NewSemanticAnalyzer() *SemanticAnalyzer {
	return &SemanticAnalyzer{CurrentScope: nil}
}

func (b *SemanticAnalyzer) error(errorCode ErrorCode, token *Token) error {
	return Error{
		ErrorCode: errorCode,
		Token:     token,
		Message:   fmt.Sprintf("%s -> %v", errorCode, token),
		Type:      SemanticErrorType,
	}
}

func (b *SemanticAnalyzer) Visit(node Ast) {
	if n, ok := node.(*Program); ok {
		b.VisitProgram(n)
		return
	}
	if n, ok := node.(*Package); ok {
		b.VisitPackage(n)
		return
	}
	if n, ok := node.(*Block); ok {
		b.VisitBlock(n)
		return
	}
	if n, ok := node.(*VarDecl); ok {
		b.VisitVarDecl(n)
		return
	}
	if n, ok := node.(*Type); ok {
		b.VisitType(n)
		return
	}
	if n, ok := node.(*BinOp); ok {
		b.VisitBinOp(n)
		return
	}
	if n, ok := node.(*Number); ok {
		b.VisitNumber(n)
		return
	}
	if n, ok := node.(*String); ok {
		b.VisitString(n)
		return
	}
	if n, ok := node.(*Boolean); ok {
		b.VisitBoolean(n)
		return
	}
	if n, ok := node.(*List); ok {
		b.VisitList(n)
		return
	}
	if n, ok := node.(*Dict); ok {
		b.VisitDict(n)
		return
	}
	if n, ok := node.(*Message); ok {
		b.VisitMessage(n)
		return
	}
	if n, ok := node.(*UnaryOp); ok {
		b.VisitUnaryOp(n)
		return
	}
	if n, ok := node.(*Compound); ok {
		b.VisitCompound(n)
		return
	}
	if n, ok := node.(*Assign); ok {
		b.VisitAssign(n)
		return
	}
	if n, ok := node.(*Var); ok {
		b.VisitVar(n)
		return
	}
	if n, ok := node.(*NoOp); ok {
		b.VisitNoOp(n)
		return
	}
	if n, ok := node.(*FunctionDecl); ok {
		b.VisitFunctionDecl(n)
		return
	}
	if n, ok := node.(*FunctionCall); ok {
		b.VisitFunctionCall(n)
		return
	}
	if n, ok := node.(*FunctionRef); ok {
		b.VisitFunctionRef(n)
		return
	}
	if n, ok := node.(*Return); ok {
		b.VisitReturn(n)
		return
	}
	if n, ok := node.(*Print); ok {
		b.VisitPrint(n)
		return
	}
	if n, ok := node.(*While); ok {
		b.VisitWhile(n)
		return
	}
	if n, ok := node.(*If); ok {
		b.VisitIf(n)
		return
	}
	if n, ok := node.(*Logical); ok {
		b.VisitLogical(n)
		return
	}
}

func (b *SemanticAnalyzer) VisitProgram(node *Program) {
	log.Println("ENTER scope: global")
	globalScope := NewScopedSymbolTable("global", 1, b.CurrentScope)
	b.CurrentScope = globalScope

	// builtin function
	for _, f := range functions {
		b.CurrentScope.Insert(f)
	}
	for _, f := range iteration {
		b.CurrentScope.Insert(f)
	}

	// import package
	for _, p := range node.Packages {
		b.Visit(p)
	}

	// visit subtree
	b.Visit(node.Block)

	log.Println(globalScope.String())

	b.CurrentScope = b.CurrentScope.EnclosingScope
	log.Println("LEAVE scope: global")
}

func (b *SemanticAnalyzer) VisitPackage(node *Package) {
	log.Println("Import package:", node.Name)
	for _, call := range packages[node.Name] {
		b.CurrentScope.Insert(call)
	}
}

func (b *SemanticAnalyzer) VisitBlock(node *Block) {
	for _, declaration := range node.Declarations {
		for _, decl := range declaration {
			b.Visit(decl)
		}
	}
	b.Visit(node.CompoundStatement)
}

func (b *SemanticAnalyzer) VisitVarDecl(node *VarDecl) {
	typeName := node.TypeNode.(*Type).Value.(string)
	typeSymbol := b.CurrentScope.Lookup(typeName, false)
	varName := node.VarNode.(*Var).Value.(string)
	varSymbol := NewVarSymbol(varName, typeSymbol)
	if b.CurrentScope.Lookup(varName, true) != nil {
		panic(b.error(DuplicateId, node.VarNode.(*Var).Token))
	}
	b.CurrentScope.Insert(varSymbol)
}

func (b *SemanticAnalyzer) VisitType(node *Type) {
	// pass
}

func (b *SemanticAnalyzer) VisitBinOp(node *BinOp) {
	b.Visit(node.Left)
	b.Visit(node.Right)
}

func (b *SemanticAnalyzer) VisitNumber(node *Number) {
	// pass
}

func (b *SemanticAnalyzer) VisitString(node *String) {
	// pass
}
func (b *SemanticAnalyzer) VisitMessage(node *Message) {
	// pass
}

func (b *SemanticAnalyzer) VisitBoolean(node *Boolean) {
	// pass
}

func (b *SemanticAnalyzer) VisitList(node *List) {
	for _, item := range node.Value {
		b.Visit(item)
	}
}

func (b *SemanticAnalyzer) VisitDict(node *Dict) {
	for _, item := range node.Value {
		b.Visit(item)
	}
}

func (b *SemanticAnalyzer) VisitUnaryOp(node *UnaryOp) {
	// pass
}

func (b *SemanticAnalyzer) VisitCompound(node *Compound) {
	for _, child := range node.Children {
		b.Visit(child)
	}
}

func (b *SemanticAnalyzer) VisitAssign(node *Assign) {
	b.Visit(node.Right)
	b.Visit(node.Left)
}

func (b *SemanticAnalyzer) VisitVar(node *Var) {
	varName := node.Value.(string)
	varSymbol := b.CurrentScope.Lookup(varName, false)

	if varSymbol == nil {
		panic(b.error(IdNotFound, node.Token))
	}
}

func (b *SemanticAnalyzer) VisitNoOp(node *NoOp) {
	// pass
}

func (b *SemanticAnalyzer) VisitFunctionDecl(node *FunctionDecl) {
	funcName := node.FuncName
	funcSymbol := NewFunctionSymbol(funcName)
	b.CurrentScope.Insert(funcSymbol)

	log.Printf("ENTER scope: %s\n", funcName)
	functionScope := NewScopedSymbolTable(funcName, b.CurrentScope.ScopeLevel+1, b.CurrentScope)
	b.CurrentScope = functionScope

	for _, param := range node.FormalParams {
		paramType := b.CurrentScope.Lookup(param.(*Param).TypeNode.(*Type).Value.(string), false)
		paramName := param.(*Param).VarNode.(*Var).Value.(string)
		varSymbol := NewVarSymbol(paramName, paramType)
		b.CurrentScope.Insert(varSymbol)
		funcSymbol.FormalParams = append(funcSymbol.FormalParams, varSymbol)
	}

	b.Visit(node.BlockNode)

	log.Println(functionScope.String())

	b.CurrentScope = b.CurrentScope.EnclosingScope
	log.Printf("LEAVE scope: %s\n", funcName)

	funcSymbol.BlockAst = node.BlockNode
	funcSymbol.ReturnType = node.ReturnType
}

func (b *SemanticAnalyzer) VisitFunctionCall(node *FunctionCall) {
	var funcName string
	if node.PackageName != "" {
		funcName = fmt.Sprintf("%s.%s", node.PackageName, node.FuncName)
	} else {
		funcName = node.FuncName
	}
	funcSymbol := b.CurrentScope.Lookup(funcName, false)
	var formalParams []Ast
	if funcSymbol != nil {
		formalParams = funcSymbol.(*FunctionSymbol).FormalParams
	} else {
		// builtin
		funcSymbol = b.CurrentScope.Lookup(fmt.Sprintf("builtin.%s", node.FuncName), false)
		if funcSymbol == nil {
			panic(b.error(UndefinedFunction, node.Token))
		}
	}
	actualParams := node.ActualParams

	if funcSymbol.(*FunctionSymbol).Package == "" {
		if len(actualParams) != len(formalParams) {
			panic(b.error(WrongParamsNum, node.Token))
		}
	} else {
		_ = funcSymbol.(*FunctionSymbol).Package
		// TODO
	}

	for _, paramNode := range node.ActualParams {
		b.Visit(paramNode)
	}

	node.FuncSymbol = funcSymbol
}

func (b *SemanticAnalyzer) VisitFunctionRef(node *FunctionRef) {
	var funcName string
	if node.PackageName != "" {
		funcName = fmt.Sprintf("%s.%s", node.PackageName, node.FuncName)
	} else {
		funcName = node.FuncName
	}
	funcSymbol := b.CurrentScope.Lookup(funcName, false)
	if funcSymbol != nil {
		// pass
	} else {
		funcSymbol = b.CurrentScope.Lookup(fmt.Sprintf("builtin.%s", node.FuncName), false)
		if funcSymbol == nil {
			panic(b.error(UndefinedFunction, node.Token))
		}
	}
}

func (b *SemanticAnalyzer) VisitReturn(node *Return) {
	b.Visit(node.Statement)
}

func (b *SemanticAnalyzer) VisitPrint(node *Print) {
	b.Visit(node.Statement)
}

func (b *SemanticAnalyzer) VisitWhile(node *While) {
	// TODO scope
	for _, node := range node.DoBranch {
		b.Visit(node)
	}
}

func (b *SemanticAnalyzer) VisitIf(node *If) {
	// TODO scope
	for _, node := range node.ThenBranch {
		b.Visit(node)
	}
	for _, node := range node.ElseBranch {
		b.Visit(node)
	}
}

func (b *SemanticAnalyzer) VisitLogical(node *Logical) {
	b.Visit(node.Left)
	b.Visit(node.Right)
}
