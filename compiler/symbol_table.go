package compiler

type SymbolScope string

const (
	GlobalScope   SymbolScope = "GLOBAL"
	LocalScope    SymbolScope = "LOCAL"
	BuiltinScope  SymbolScope = "BUILTIN"
	FreeScope     SymbolScope = "FREE"
	FunctionScope SymbolScope = "FUNCTION"
)

type Symbol struct {
	Name    string
	Scope   SymbolScope
	Index   int
	IsConst bool
	File    string
}

type SymbolTable struct {
	Outer       *SymbolTable
	isBlock     bool
	CurrentFile string

	store          map[string]Symbol
	numDefinitions int

	FreeSymbols []Symbol
}

func NewSymbolTable() *SymbolTable {
	s := make(map[string]Symbol)
	return &SymbolTable{
		store:       s,
		FreeSymbols: []Symbol{},
	}
}

func NewEnclosedSymbolTable(outer *SymbolTable) *SymbolTable {
	s := NewSymbolTable()
	s.Outer = outer
	s.isBlock = false
	if outer != nil {
		s.CurrentFile = outer.CurrentFile
	}
	return s
}

func NewEnclosedBlockSymbolTable(outer *SymbolTable) *SymbolTable {
	s := NewSymbolTable()
	s.Outer = outer
	s.isBlock = true
	if outer != nil {
		s.CurrentFile = outer.CurrentFile
	}
	return s
}

func (s *SymbolTable) Store() map[string]Symbol {
	return s.store
}

func (s *SymbolTable) enclosingFunctionScope() *SymbolTable {
	curr := s
	for curr != nil {
		if !curr.isBlock {
			return curr
		}
		curr = curr.Outer
	}
	return nil
}

func (s *SymbolTable) Define(name string, isConst bool) Symbol {
	fnScope := s.enclosingFunctionScope()
	var scope SymbolScope
	var index int

	if fnScope == nil || fnScope.Outer == nil {
		scope = GlobalScope
		if fnScope != nil {
			index = fnScope.numDefinitions
			fnScope.numDefinitions++
		} else {
			index = s.numDefinitions
			s.numDefinitions++
		}
	} else {
		scope = LocalScope
		index = fnScope.numDefinitions
		fnScope.numDefinitions++
	}

	symbol := Symbol{Name: name, Scope: scope, Index: index, IsConst: isConst, File: s.CurrentFile}
	s.store[name] = symbol
	return symbol
}

func (s *SymbolTable) DefineGlobal(name string, isConst bool) Symbol {
	root := s
	for root.Outer != nil {
		root = root.Outer
	}
	if sym, ok := root.store[name]; ok {
		return sym
	}
	sym := Symbol{
		Name:    name,
		Scope:   GlobalScope,
		Index:   root.numDefinitions,
		IsConst: isConst,
		File:    root.CurrentFile,
	}
	root.numDefinitions++
	root.store[name] = sym
	return sym
}

func (s *SymbolTable) DefineBuiltin(index int, name string) Symbol {
	symbol := Symbol{Name: name, Index: index, Scope: BuiltinScope}
	s.store[name] = symbol
	return symbol
}

func (s *SymbolTable) DefineFunctionName(name string) Symbol {
	symbol := Symbol{Name: name, Index: 0, Scope: FunctionScope, File: s.CurrentFile}
	s.store[name] = symbol
	return symbol
}

func (s *SymbolTable) Resolve(name string) (Symbol, bool) {
	obj, ok := s.store[name]
	if ok {
		return obj, true
	}

	if s.Outer == nil {
		return obj, false
	}

	obj, ok = s.Outer.Resolve(name)
	if !ok {
		return obj, false
	}

	if obj.Scope == GlobalScope || obj.Scope == BuiltinScope {
		return obj, true
	}

	if s.isBlock {
		return obj, true
	}

	free := s.defineFree(obj)
	return free, true
}

func (s *SymbolTable) defineFree(original Symbol) Symbol {
	s.FreeSymbols = append(s.FreeSymbols, original)

	symbol := Symbol{
		Name:    original.Name,
		Index:   len(s.FreeSymbols) - 1,
		Scope:   FreeScope,
		IsConst: original.IsConst,
		File:    original.File,
	}

	s.store[original.Name] = symbol
	return symbol
}
