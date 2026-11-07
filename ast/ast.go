package ast

import (
	"bramandscript/token"
	"bytes"
	"fmt"
	"math/big"
	"strings"
)

type Node interface {
	TokenLiteral() string
	String() string
}

type Statement interface {
	Node
	statementNode()
}

type Expression interface {
	Node
	expressionNode()
}

type Program struct {
	Statements []Statement
}

func (p *Program) TokenLiteral() string {
	if len(p.Statements) > 0 {
		return p.Statements[0].TokenLiteral()
	}
	return ""
}

func (p *Program) String() string {
	var out bytes.Buffer
	for _, s := range p.Statements {
		out.WriteString(s.String())
	}
	return out.String()
}

// ---------------- Statements ----------------

type LetStatement struct {
	Token   token.Token // token.LET
	Name    *Identifier
	Pattern Node // Pattern if destructuring (ArrayPattern or HashPattern)
	Value   Expression
}

func (ls *LetStatement) statementNode()       {}
func (ls *LetStatement) TokenLiteral() string { return ls.Token.Literal }
func (ls *LetStatement) String() string {
	var out bytes.Buffer
	out.WriteString(ls.TokenLiteral() + " ")
	if ls.Name != nil {
		out.WriteString(ls.Name.String())
	} else if ls.Pattern != nil {
		out.WriteString(ls.Pattern.String())
	}
	out.WriteString(" = ")
	if ls.Value != nil {
		out.WriteString(ls.Value.String())
	}
	out.WriteString(";")
	return out.String()
}

type ConstStatement struct {
	Token   token.Token // token.CONST
	Name    *Identifier
	Pattern Node // Pattern if destructuring (ArrayPattern or HashPattern)
	Value   Expression
}

func (cs *ConstStatement) statementNode()       {}
func (cs *ConstStatement) TokenLiteral() string { return cs.Token.Literal }
func (cs *ConstStatement) String() string {
	var out bytes.Buffer
	out.WriteString(cs.TokenLiteral() + " ")
	if cs.Name != nil {
		out.WriteString(cs.Name.String())
	} else if cs.Pattern != nil {
		out.WriteString(cs.Pattern.String())
	}
	out.WriteString(" = ")
	if cs.Value != nil {
		out.WriteString(cs.Value.String())
	}
	out.WriteString(";")
	return out.String()
}

type ReassignStatement struct {
	Token    token.Token // '=', '+=', etc.
	Name     *Identifier
	Operator string
	Value    Expression
}

func (rs *ReassignStatement) statementNode()       {}
func (rs *ReassignStatement) TokenLiteral() string { return rs.Token.Literal }
func (rs *ReassignStatement) String() string {
	var out bytes.Buffer
	out.WriteString(rs.Name.String() + " " + rs.Operator + " ")
	if rs.Value != nil {
		out.WriteString(rs.Value.String())
	}
	out.WriteString(";")
	return out.String()
}

type ReturnStatement struct {
	Token       token.Token // token.RETURN
	ReturnValue Expression
}

func (rs *ReturnStatement) statementNode()       {}
func (rs *ReturnStatement) TokenLiteral() string { return rs.Token.Literal }
func (rs *ReturnStatement) String() string {
	var out bytes.Buffer
	out.WriteString(rs.TokenLiteral() + " ")
	if rs.ReturnValue != nil {
		out.WriteString(rs.ReturnValue.String())
	}
	out.WriteString(";")
	return out.String()
}

type IndexAssignStatement struct {
	Token    token.Token // '=', '+=', etc.
	Left     Expression
	Index    Expression
	Operator string
	Value    Expression
}

func (ias *IndexAssignStatement) statementNode()       {}
func (ias *IndexAssignStatement) TokenLiteral() string { return ias.Token.Literal }
func (ias *IndexAssignStatement) String() string {
	op := ias.Operator
	if op == "" {
		op = "="
	}
	return ias.Left.String() + "[" + ias.Index.String() + "] " + op + " " + ias.Value.String() + ";"
}

type ForInStatement struct {
	Token    token.Token // token.FOR
	Item     *Identifier
	Index    *Identifier // optional
	Iterable Expression
	Body     *BlockStatement
}

func (fis *ForInStatement) statementNode()       {}
func (fis *ForInStatement) TokenLiteral() string { return fis.Token.Literal }
func (fis *ForInStatement) String() string {
	var out bytes.Buffer
	out.WriteString("for ")
	if fis.Index != nil {
		out.WriteString(fis.Index.String() + ", ")
	}
	out.WriteString(fis.Item.String() + " in " + fis.Iterable.String() + " ")
	out.WriteString(fis.Body.String())
	return out.String()
}

type TryCatchStatement struct {
	Token        token.Token // token.TRY
	Block        *BlockStatement
	CatchVar     *Identifier // optional error variable e.g. err
	CatchBlock   *BlockStatement
	FinallyBlock *BlockStatement // optional
}

func (tcs *TryCatchStatement) statementNode()       {}
func (tcs *TryCatchStatement) TokenLiteral() string { return tcs.Token.Literal }
func (tcs *TryCatchStatement) String() string {
	var out bytes.Buffer
	out.WriteString("try ")
	if tcs.Block != nil {
		out.WriteString(tcs.Block.String())
	}
	if tcs.CatchBlock != nil {
		out.WriteString(" catch ")
		if tcs.CatchVar != nil {
			out.WriteString(tcs.CatchVar.String() + " ")
		}
		out.WriteString(tcs.CatchBlock.String())
	}
	if tcs.FinallyBlock != nil {
		out.WriteString(" finally ")
		out.WriteString(tcs.FinallyBlock.String())
	}
	return out.String()
}

type ThrowStatement struct {
	Token token.Token // token.THROW
	Value Expression
}

func (ts *ThrowStatement) statementNode()       {}
func (ts *ThrowStatement) TokenLiteral() string { return ts.Token.Literal }
func (ts *ThrowStatement) String() string {
	var out bytes.Buffer
	out.WriteString("throw ")
	if ts.Value != nil {
		out.WriteString(ts.Value.String())
	}
	out.WriteString(";")
	return out.String()
}

type DeferStatement struct {
	Token token.Token // token.DEFER
	Call  Expression
}

func (ds *DeferStatement) statementNode()       {}
func (ds *DeferStatement) TokenLiteral() string { return ds.Token.Literal }
func (ds *DeferStatement) String() string {
	var out bytes.Buffer
	out.WriteString("defer ")
	if ds.Call != nil {
		out.WriteString(ds.Call.String())
	}
	out.WriteString(";")
	return out.String()
}

type ImportStatement struct {
	Token token.Token // token.IMPORT
	Path  string
	Alias string
}

func (is *ImportStatement) statementNode()       {}
func (is *ImportStatement) TokenLiteral() string { return is.Token.Literal }
func (is *ImportStatement) String() string {
	if is.Alias != "" {
		return "import \"" + is.Path + "\" as " + is.Alias + ";"
	}
	return "import \"" + is.Path + "\";"
}

type StructStatement struct {
	Token   token.Token // token.STRUCT
	Name    *Identifier
	Embeds  []*Identifier
	Fields  []*Identifier
	Methods []*FunctionLiteral
}

func (ss *StructStatement) statementNode()       {}
func (ss *StructStatement) TokenLiteral() string { return ss.Token.Literal }
func (ss *StructStatement) String() string {
	var out bytes.Buffer
	out.WriteString("struct ")
	out.WriteString(ss.Name.String())
	out.WriteString(" { ")
	for i, f := range ss.Fields {
		if i > 0 {
			out.WriteString(", ")
		}
		out.WriteString(f.String())
	}
	for _, m := range ss.Methods {
		out.WriteString(" ")
		out.WriteString(m.String())
	}
	out.WriteString(" }")
	return out.String()
}

type EnumEntry struct {
	Name  *Identifier
	Value Expression
}

type EnumStatement struct {
	Token   token.Token // token.ENUM
	Name    *Identifier
	Entries []*EnumEntry
}

func (es *EnumStatement) statementNode()       {}
func (es *EnumStatement) TokenLiteral() string { return es.Token.Literal }
func (es *EnumStatement) String() string {
	var out bytes.Buffer
	out.WriteString("enum ")
	out.WriteString(es.Name.String())
	out.WriteString(" { ")
	for i, entry := range es.Entries {
		if i > 0 {
			out.WriteString(", ")
		}
		out.WriteString(entry.Name.String())
		if entry.Value != nil {
			out.WriteString(" = ")
			out.WriteString(entry.Value.String())
		}
	}
	out.WriteString(" }")
	return out.String()
}

type ExpressionStatement struct {
	Token      token.Token // first token of expression
	Expression Expression
}

func (es *ExpressionStatement) statementNode()       {}
func (es *ExpressionStatement) TokenLiteral() string { return es.Token.Literal }
func (es *ExpressionStatement) String() string {
	if es.Expression != nil {
		return es.Expression.String()
	}
	return ""
}

type BlockStatement struct {
	Token      token.Token // token.LBRACE
	Statements []Statement
}

func (bs *BlockStatement) statementNode()       {}
func (bs *BlockStatement) TokenLiteral() string { return bs.Token.Literal }
func (bs *BlockStatement) String() string {
	var out bytes.Buffer
	for _, s := range bs.Statements {
		out.WriteString(s.String())
	}
	return out.String()
}

type WhileStatement struct {
	Token     token.Token // token.WHILE
	Condition Expression
	Body      *BlockStatement
}

func (ws *WhileStatement) statementNode()       {}
func (ws *WhileStatement) TokenLiteral() string { return ws.Token.Literal }
func (ws *WhileStatement) String() string {
	var out bytes.Buffer
	out.WriteString("while ")
	if ws.Condition != nil {
		out.WriteString(ws.Condition.String())
	}
	out.WriteString(" ")
	if ws.Body != nil {
		out.WriteString(ws.Body.String())
	}
	return out.String()
}

type BreakStatement struct {
	Token token.Token // token.BREAK
}

func (bs *BreakStatement) statementNode()       {}
func (bs *BreakStatement) TokenLiteral() string { return bs.Token.Literal }
func (bs *BreakStatement) String() string       { return "break;" }

type ContinueStatement struct {
	Token token.Token // token.CONTINUE
}

func (cs *ContinueStatement) statementNode()       {}
func (cs *ContinueStatement) TokenLiteral() string { return cs.Token.Literal }
func (cs *ContinueStatement) String() string       { return "continue;" }

// ---------------- Expressions ----------------

type Identifier struct {
	Token token.Token // token.IDENT
	Value string
}

func (i *Identifier) expressionNode()      {}
func (i *Identifier) TokenLiteral() string { return i.Token.Literal }
func (i *Identifier) String() string       { return i.Value }

type IntegerLiteral struct {
	Token    token.Token
	Value    int64
	BigValue *big.Int
}

func (il *IntegerLiteral) expressionNode()      {}
func (il *IntegerLiteral) TokenLiteral() string { return il.Token.Literal }
func (il *IntegerLiteral) String() string       { return il.Token.Literal }

type FloatLiteral struct {
	Token token.Token
	Value float64
}

func (fl *FloatLiteral) expressionNode()      {}
func (fl *FloatLiteral) TokenLiteral() string { return fl.Token.Literal }
func (fl *FloatLiteral) String() string       { return fl.Token.Literal }

type StringLiteral struct {
	Token token.Token
	Value string
}

func (sl *StringLiteral) expressionNode()      {}
func (sl *StringLiteral) TokenLiteral() string { return sl.Token.Literal }
func (sl *StringLiteral) String() string       { return sl.Token.Literal }

type BooleanLiteral struct {
	Token token.Token
	Value bool
}

func (b *BooleanLiteral) expressionNode()      {}
func (b *BooleanLiteral) TokenLiteral() string { return b.Token.Literal }
func (b *BooleanLiteral) String() string       { return b.Token.Literal }

type NullLiteral struct {
	Token token.Token
}

func (n *NullLiteral) expressionNode()      {}
func (n *NullLiteral) TokenLiteral() string { return n.Token.Literal }
func (n *NullLiteral) String() string       { return "null" }

type PrefixExpression struct {
	Token    token.Token // '!', '-'
	Operator string
	Right    Expression
}

func (pe *PrefixExpression) expressionNode()      {}
func (pe *PrefixExpression) TokenLiteral() string { return pe.Token.Literal }
func (pe *PrefixExpression) String() string {
	var out bytes.Buffer
	out.WriteString("(")
	out.WriteString(pe.Operator)
	out.WriteString(pe.Right.String())
	out.WriteString(")")
	return out.String()
}

type InfixExpression struct {
	Token    token.Token
	Left     Expression
	Operator string
	Right    Expression
}

func (oe *InfixExpression) expressionNode()      {}
func (oe *InfixExpression) TokenLiteral() string { return oe.Token.Literal }
func (oe *InfixExpression) String() string {
	var out bytes.Buffer
	out.WriteString("(")
	out.WriteString(oe.Left.String())
	out.WriteString(" " + oe.Operator + " ")
	out.WriteString(oe.Right.String())
	out.WriteString(")")
	return out.String()
}

type IfExpression struct {
	Token       token.Token // token.IF
	Condition   Expression
	Consequence *BlockStatement
	Alternative *BlockStatement
}

func (ie *IfExpression) expressionNode()      {}
func (ie *IfExpression) TokenLiteral() string { return ie.Token.Literal }
func (ie *IfExpression) String() string {
	var out bytes.Buffer
	out.WriteString("if")
	out.WriteString(ie.Condition.String())
	out.WriteString(" ")
	out.WriteString(ie.Consequence.String())
	if ie.Alternative != nil {
		out.WriteString("else ")
		out.WriteString(ie.Alternative.String())
	}
	return out.String()
}

type FunctionLiteral struct {
	Token      token.Token // token.FUNCTION
	Parameters []*Identifier
	Defaults   map[string]Expression
	Body       *BlockStatement
	Name       string // optional named function: fn add(a, b) {...}
}

func (fl *FunctionLiteral) expressionNode()      {}
func (fl *FunctionLiteral) TokenLiteral() string { return fl.Token.Literal }
func (fl *FunctionLiteral) String() string {
	var out bytes.Buffer
	var params []string
	for _, p := range fl.Parameters {
		if fl.Defaults != nil {
			if defVal, ok := fl.Defaults[p.Value]; ok {
				params = append(params, p.String()+" = "+defVal.String())
				continue
			}
		}
		params = append(params, p.String())
	}
	out.WriteString(fl.TokenLiteral())
	if fl.Name != "" {
		out.WriteString(" " + fl.Name)
	}
	out.WriteString("(")
	out.WriteString(strings.Join(params, ", "))
	out.WriteString(") ")
	out.WriteString(fl.Body.String())
	return out.String()
}

type CallExpression struct {
	Token     token.Token // '('
	Function  Expression  // Identifier or FunctionLiteral
	Arguments []Expression
}

func (ce *CallExpression) expressionNode()      {}
func (ce *CallExpression) TokenLiteral() string { return ce.Token.Literal }
func (ce *CallExpression) String() string {
	var out bytes.Buffer
	var args []string
	for _, a := range ce.Arguments {
		args = append(args, a.String())
	}
	out.WriteString(ce.Function.String())
	out.WriteString("(")
	out.WriteString(strings.Join(args, ", "))
	out.WriteString(")")
	return out.String()
}

type ArrayLiteral struct {
	Token    token.Token // '['
	Elements []Expression
}

func (al *ArrayLiteral) expressionNode()      {}
func (al *ArrayLiteral) TokenLiteral() string { return al.Token.Literal }
func (al *ArrayLiteral) String() string {
	var out bytes.Buffer
	var elements []string
	for _, el := range al.Elements {
		elements = append(elements, el.String())
	}
	out.WriteString("[")
	out.WriteString(strings.Join(elements, ", "))
	out.WriteString("]")
	return out.String()
}

type HashEntry struct {
	Key   Expression
	Value Expression // nil if spread entry
}

type HashLiteral struct {
	Token   token.Token // '{'
	Pairs   map[Expression]Expression
	Entries []HashEntry
}

func (hl *HashLiteral) expressionNode()      {}
func (hl *HashLiteral) TokenLiteral() string { return hl.Token.Literal }
func (hl *HashLiteral) String() string {
	var out bytes.Buffer
	var pairs []string
	if len(hl.Entries) > 0 {
		for _, entry := range hl.Entries {
			if entry.Value == nil {
				pairs = append(pairs, entry.Key.String())
			} else {
				pairs = append(pairs, entry.Key.String()+": "+entry.Value.String())
			}
		}
	} else {
		for key, value := range hl.Pairs {
			if value == nil {
				pairs = append(pairs, key.String())
			} else {
				pairs = append(pairs, key.String()+": "+value.String())
			}
		}
	}
	out.WriteString("{")
	out.WriteString(strings.Join(pairs, ", "))
	out.WriteString("}")
	return out.String()
}

type SpreadExpression struct {
	Token token.Token // token.ELLIPSIS
	Value Expression
}

func (se *SpreadExpression) expressionNode()      {}
func (se *SpreadExpression) TokenLiteral() string { return se.Token.Literal }
func (se *SpreadExpression) String() string {
	if se.Value != nil {
		return "..." + se.Value.String()
	}
	return "..."
}

type IndexExpression struct {
	Token token.Token // '['
	Left  Expression
	Index Expression
}

func (ie *IndexExpression) expressionNode()      {}
func (ie *IndexExpression) TokenLiteral() string { return ie.Token.Literal }
func (ie *IndexExpression) String() string {
	var out bytes.Buffer
	out.WriteString("(")
	out.WriteString(ie.Left.String())
	out.WriteString("[")
	out.WriteString(ie.Index.String())
	out.WriteString("])")
	return out.String()
}

type SliceExpression struct {
	Token token.Token // '['
	Left  Expression
	Start Expression
	End   Expression
	Step  Expression
}

func (se *SliceExpression) expressionNode()      {}
func (se *SliceExpression) TokenLiteral() string { return se.Token.Literal }
func (se *SliceExpression) String() string {
	var out bytes.Buffer
	out.WriteString("(")
	out.WriteString(se.Left.String())
	out.WriteString("[")
	if se.Start != nil {
		out.WriteString(se.Start.String())
	}
	out.WriteString(":")
	if se.End != nil {
		out.WriteString(se.End.String())
	}
	if se.Step != nil {
		out.WriteString(":")
		out.WriteString(se.Step.String())
	}
	out.WriteString("])")
	return out.String()
}

type DotExpression struct {
	Token token.Token // token.DOT
	Left  Expression
	Field *Identifier
}

func (de *DotExpression) expressionNode()      {}
func (de *DotExpression) TokenLiteral() string { return de.Token.Literal }
func (de *DotExpression) String() string {
	return de.Left.String() + "." + de.Field.String()
}

type MatchClause struct {
	Values []Expression // nil indicates default
	Body   *BlockStatement
}

type MatchExpression struct {
	Token   token.Token // token.MATCH
	Subject Expression
	Clauses []*MatchClause
}

func (me *MatchExpression) expressionNode()      {}
func (me *MatchExpression) TokenLiteral() string { return me.Token.Literal }
func (me *MatchExpression) String() string {
	var out bytes.Buffer
	out.WriteString("match ")
	if me.Subject != nil {
		out.WriteString(me.Subject.String())
	}
	out.WriteString(" { ")
	for _, c := range me.Clauses {
		if len(c.Values) == 0 {
			out.WriteString("default => " + c.Body.String() + " ")
		} else {
			out.WriteString("case ... => " + c.Body.String() + " ")
		}
	}
	out.WriteString("}")
	return out.String()
}

type SpawnExpression struct {
	Token token.Token // token.SPAWN
	Call  Expression
}

func (se *SpawnExpression) expressionNode()      {}
func (se *SpawnExpression) TokenLiteral() string { return se.Token.Literal }
func (se *SpawnExpression) String() string {
	if se.Call != nil {
		return "spawn " + se.Call.String()
	}
	return "spawn"
}

type PipeExpression struct {
	Token token.Token // token.PIPE_OP
	Left  Expression
	Right Expression
}

func (pe *PipeExpression) expressionNode()      {}
func (pe *PipeExpression) TokenLiteral() string { return pe.Token.Literal }
func (pe *PipeExpression) String() string {
	return "(" + pe.Left.String() + " |> " + pe.Right.String() + ")"
}

type ArrayPattern struct {
	Token    token.Token // token.LBRACKET
	Elements []Expression // *Identifier or *RestPattern
}

func (ap *ArrayPattern) expressionNode()      {}
func (ap *ArrayPattern) TokenLiteral() string { return ap.Token.Literal }
func (ap *ArrayPattern) String() string {
	var out bytes.Buffer
	var elements []string
	for _, el := range ap.Elements {
		elements = append(elements, el.String())
	}
	out.WriteString("[")
	out.WriteString(strings.Join(elements, ", "))
	out.WriteString("]")
	return out.String()
}

type RestPattern struct {
	Token  token.Token // token.ELLIPSIS
	Target *Identifier
}

func (rp *RestPattern) expressionNode()      {}
func (rp *RestPattern) TokenLiteral() string { return rp.Token.Literal }
func (rp *RestPattern) String() string {
	if rp.Target != nil {
		return "..." + rp.Target.String()
	}
	return "..."
}

type HashPattern struct {
	Token token.Token // token.LBRACE
	Keys  []*Identifier
}

func (hp *HashPattern) expressionNode()      {}
func (hp *HashPattern) TokenLiteral() string { return hp.Token.Literal }
func (hp *HashPattern) String() string {
	var out bytes.Buffer
	var keys []string
	for _, k := range hp.Keys {
		keys = append(keys, k.String())
	}
	out.WriteString("{ ")
	out.WriteString(strings.Join(keys, ", "))
	out.WriteString(" }")
	return out.String()
}

type YieldExpression struct {
	Token token.Token // token.YIELD
	Value Expression
}

func (ye *YieldExpression) expressionNode()      {}
func (ye *YieldExpression) TokenLiteral() string { return ye.Token.Literal }
func (ye *YieldExpression) String() string {
	if ye.Value != nil {
		return "yield " + ye.Value.String()
	}
	return "yield"
}

// ---------------- Enterprise Language AST Nodes ----------------

type TernaryExpression struct {
	Token       token.Token // token.QUESTION '?'
	Condition   Expression
	Consequence Expression
	Alternative Expression
}

func (te *TernaryExpression) expressionNode()      {}
func (te *TernaryExpression) TokenLiteral() string { return te.Token.Literal }
func (te *TernaryExpression) String() string {
	return fmt.Sprintf("(%s ? %s : %s)", te.Condition.String(), te.Consequence.String(), te.Alternative.String())
}

type NullCoalesceExpression struct {
	Token token.Token // token.COALESCE '??'
	Left  Expression
	Right Expression
}

func (nce *NullCoalesceExpression) expressionNode()      {}
func (nce *NullCoalesceExpression) TokenLiteral() string { return nce.Token.Literal }
func (nce *NullCoalesceExpression) String() string {
	return fmt.Sprintf("(%s ?? %s)", nce.Left.String(), nce.Right.String())
}

type OptionalDotExpression struct {
	Token token.Token // token.QUESTION_DOT '?.'
	Left  Expression
	Field *Identifier
}

func (ode *OptionalDotExpression) expressionNode()      {}
func (ode *OptionalDotExpression) TokenLiteral() string { return ode.Token.Literal }
func (ode *OptionalDotExpression) String() string {
	return ode.Left.String() + "?." + ode.Field.String()
}

type OptionalIndexExpression struct {
	Token token.Token // token.QUESTION_DOT '?.'
	Left  Expression
	Index Expression
}

func (oie *OptionalIndexExpression) expressionNode()      {}
func (oie *OptionalIndexExpression) TokenLiteral() string { return oie.Token.Literal }
func (oie *OptionalIndexExpression) String() string {
	return oie.Left.String() + "?." + "[" + oie.Index.String() + "]"
}

type OptionalCallExpression struct {
	Token     token.Token // token.QUESTION_DOT '?.'
	Function  Expression
	Arguments []Expression
}

func (oce *OptionalCallExpression) expressionNode()      {}
func (oce *OptionalCallExpression) TokenLiteral() string { return oce.Token.Literal }
func (oce *OptionalCallExpression) String() string {
	var out bytes.Buffer
	var args []string
	for _, a := range oce.Arguments {
		args = append(args, a.String())
	}
	out.WriteString(oce.Function.String())
	out.WriteString("?.(")
	out.WriteString(strings.Join(args, ", "))
	out.WriteString(")")
	return out.String()
}

type InterpolatedStringLiteral struct {
	Token token.Token // token.FORMAT_STRING
	Parts []Expression
}

func (isl *InterpolatedStringLiteral) expressionNode()      {}
func (isl *InterpolatedStringLiteral) TokenLiteral() string { return isl.Token.Literal }
func (isl *InterpolatedStringLiteral) String() string {
	var out bytes.Buffer
	out.WriteString("f\"")
	for _, p := range isl.Parts {
		out.WriteString(p.String())
	}
	out.WriteString("\"")
	return out.String()
}
