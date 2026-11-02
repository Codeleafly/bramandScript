package object

import (
	"fmt"
	"math"
	"math/big"
	"strings"
)

// ---------------- BigInt ----------------

type BigInt struct {
	Value *big.Int
}

func (b *BigInt) Type() ObjectType { return BIG_INT_OBJ }
func (b *BigInt) Inspect() string  { return b.Value.String() }

func (b *BigInt) Add(other *BigInt) *BigInt {
	res := new(big.Int).Add(b.Value, other.Value)
	return &BigInt{Value: res}
}

func (b *BigInt) Sub(other *BigInt) *BigInt {
	res := new(big.Int).Sub(b.Value, other.Value)
	return &BigInt{Value: res}
}

func (b *BigInt) Mul(other *BigInt) *BigInt {
	res := new(big.Int).Mul(b.Value, other.Value)
	return &BigInt{Value: res}
}

func (b *BigInt) Div(other *BigInt) (*BigInt, error) {
	if other.Value.Sign() == 0 {
		return nil, fmt.Errorf("division by zero in BigInt")
	}
	res := new(big.Int).Quo(b.Value, other.Value)
	return &BigInt{Value: res}, nil
}

func (b *BigInt) Mod(other *BigInt) (*BigInt, error) {
	if other.Value.Sign() == 0 {
		return nil, fmt.Errorf("modulo by zero in BigInt")
	}
	res := new(big.Int).Rem(b.Value, other.Value)
	return &BigInt{Value: res}, nil
}

func (b *BigInt) Pow(exp *BigInt) *BigInt {
	res := new(big.Int).Exp(b.Value, exp.Value, nil)
	return &BigInt{Value: res}
}

func (b *BigInt) Cmp(other *BigInt) int {
	return b.Value.Cmp(other.Value)
}

func (b *BigInt) Neg() *BigInt {
	res := new(big.Int).Neg(b.Value)
	return &BigInt{Value: res}
}

func (b *BigInt) Abs() *BigInt {
	res := new(big.Int).Abs(b.Value)
	return &BigInt{Value: res}
}

// ---------------- BigFloat ----------------

type BigFloat struct {
	Value *big.Float
}

func (b *BigFloat) Type() ObjectType { return BIG_FLOAT_OBJ }
func (b *BigFloat) Inspect() string  { return b.Value.Text('g', -1) }

func (b *BigFloat) Add(other *BigFloat) *BigFloat {
	res := new(big.Float).Add(b.Value, other.Value)
	return &BigFloat{Value: res}
}

func (b *BigFloat) Sub(other *BigFloat) *BigFloat {
	res := new(big.Float).Sub(b.Value, other.Value)
	return &BigFloat{Value: res}
}

func (b *BigFloat) Mul(other *BigFloat) *BigFloat {
	res := new(big.Float).Mul(b.Value, other.Value)
	return &BigFloat{Value: res}
}

func (b *BigFloat) Div(other *BigFloat) (*BigFloat, error) {
	if other.Value.Sign() == 0 {
		return nil, fmt.Errorf("division by zero in BigFloat")
	}
	res := new(big.Float).Quo(b.Value, other.Value)
	return &BigFloat{Value: res}, nil
}

func (b *BigFloat) Sqrt() (*BigFloat, error) {
	if b.Value.Sign() < 0 {
		return nil, fmt.Errorf("square root of negative BigFloat")
	}
	res := new(big.Float).Sqrt(b.Value)
	return &BigFloat{Value: res}, nil
}

func (b *BigFloat) Cmp(other *BigFloat) int {
	return b.Value.Cmp(other.Value)
}

func (b *BigFloat) Neg() *BigFloat {
	res := new(big.Float).Neg(b.Value)
	return &BigFloat{Value: res}
}

func (b *BigFloat) Abs() *BigFloat {
	res := new(big.Float).Abs(b.Value)
	return &BigFloat{Value: res}
}

// ---------------- Vector ----------------

type Vector struct {
	Elements []float64
}

func (v *Vector) Type() ObjectType { return VECTOR_OBJ }

func (v *Vector) Inspect() string {
	var parts []string
	for _, el := range v.Elements {
		parts = append(parts, fmt.Sprintf("%g", el))
	}
	return fmt.Sprintf("vec([%s])", strings.Join(parts, ", "))
}

func (v *Vector) Dim() int {
	return len(v.Elements)
}

func (v *Vector) Dot(other *Vector) (float64, error) {
	if len(v.Elements) != len(other.Elements) {
		return 0, fmt.Errorf("vector dimensions mismatch: %d != %d", len(v.Elements), len(other.Elements))
	}
	var sum float64
	for i := range v.Elements {
		sum += v.Elements[i] * other.Elements[i]
	}
	return sum, nil
}

func (v *Vector) Cross(other *Vector) (*Vector, error) {
	if len(v.Elements) != 3 || len(other.Elements) != 3 {
		return nil, fmt.Errorf("cross product is only defined for 3D vectors (got %d and %d)", len(v.Elements), len(other.Elements))
	}
	a, b := v.Elements, other.Elements
	res := []float64{
		a[1]*b[2] - a[2]*b[1],
		a[2]*b[0] - a[0]*b[2],
		a[0]*b[1] - a[1]*b[0],
	}
	return &Vector{Elements: res}, nil
}

func (v *Vector) Norm() float64 {
	var sum float64
	for _, el := range v.Elements {
		sum += el * el
	}
	return math.Sqrt(sum)
}

func (v *Vector) Normalize() (*Vector, error) {
	norm := v.Norm()
	if norm == 0 {
		return nil, fmt.Errorf("cannot normalize zero vector")
	}
	res := make([]float64, len(v.Elements))
	for i, el := range v.Elements {
		res[i] = el / norm
	}
	return &Vector{Elements: res}, nil
}

func (v *Vector) Add(other *Vector) (*Vector, error) {
	if len(v.Elements) != len(other.Elements) {
		return nil, fmt.Errorf("vector dimensions mismatch for addition: %d != %d", len(v.Elements), len(other.Elements))
	}
	res := make([]float64, len(v.Elements))
	for i := range v.Elements {
		res[i] = v.Elements[i] + other.Elements[i]
	}
	return &Vector{Elements: res}, nil
}

func (v *Vector) Sub(other *Vector) (*Vector, error) {
	if len(v.Elements) != len(other.Elements) {
		return nil, fmt.Errorf("vector dimensions mismatch for subtraction: %d != %d", len(v.Elements), len(other.Elements))
	}
	res := make([]float64, len(v.Elements))
	for i := range v.Elements {
		res[i] = v.Elements[i] - other.Elements[i]
	}
	return &Vector{Elements: res}, nil
}

func (v *Vector) Scale(scalar float64) *Vector {
	res := make([]float64, len(v.Elements))
	for i := range v.Elements {
		res[i] = v.Elements[i] * scalar
	}
	return &Vector{Elements: res}
}

func (v *Vector) ToArray() *Array {
	elements := make([]Object, len(v.Elements))
	for i, el := range v.Elements {
		elements[i] = &Float{Value: el}
	}
	return &Array{Elements: elements}
}

// ---------------- Matrix ----------------

type Matrix struct {
	Rows int
	Cols int
	Data [][]float64
}

func NewMatrix(rows, cols int) *Matrix {
	data := make([][]float64, rows)
	for i := range data {
		data[i] = make([]float64, cols)
	}
	return &Matrix{Rows: rows, Cols: cols, Data: data}
}

func (m *Matrix) Type() ObjectType { return MATRIX_OBJ }

func (m *Matrix) Inspect() string {
	var rows []string
	for _, r := range m.Data {
		var parts []string
		for _, el := range r {
			parts = append(parts, fmt.Sprintf("%g", el))
		}
		rows = append(rows, "["+strings.Join(parts, ", ")+"]")
	}
	return fmt.Sprintf("matrix([%s])", strings.Join(rows, ", "))
}

func (m *Matrix) Get(r, c int) (float64, error) {
	if r < 0 || r >= m.Rows || c < 0 || c >= m.Cols {
		return 0, fmt.Errorf("matrix index out of bounds: (%d, %d)", r, c)
	}
	return m.Data[r][c], nil
}

func (m *Matrix) Set(r, c int, val float64) error {
	if r < 0 || r >= m.Rows || c < 0 || c >= m.Cols {
		return fmt.Errorf("matrix index out of bounds: (%d, %d)", r, c)
	}
	m.Data[r][c] = val
	return nil
}

func (m *Matrix) Add(other *Matrix) (*Matrix, error) {
	if m.Rows != other.Rows || m.Cols != other.Cols {
		return nil, fmt.Errorf("matrix dimensions mismatch for addition: (%d, %d) != (%d, %d)", m.Rows, m.Cols, other.Rows, other.Cols)
	}
	res := NewMatrix(m.Rows, m.Cols)
	for i := 0; i < m.Rows; i++ {
		for j := 0; j < m.Cols; j++ {
			res.Data[i][j] = m.Data[i][j] + other.Data[i][j]
		}
	}
	return res, nil
}

func (m *Matrix) Sub(other *Matrix) (*Matrix, error) {
	if m.Rows != other.Rows || m.Cols != other.Cols {
		return nil, fmt.Errorf("matrix dimensions mismatch for subtraction: (%d, %d) != (%d, %d)", m.Rows, m.Cols, other.Rows, other.Cols)
	}
	res := NewMatrix(m.Rows, m.Cols)
	for i := 0; i < m.Rows; i++ {
		for j := 0; j < m.Cols; j++ {
			res.Data[i][j] = m.Data[i][j] - other.Data[i][j]
		}
	}
	return res, nil
}

func (m *Matrix) Scale(scalar float64) *Matrix {
	res := NewMatrix(m.Rows, m.Cols)
	for i := 0; i < m.Rows; i++ {
		for j := 0; j < m.Cols; j++ {
			res.Data[i][j] = m.Data[i][j] * scalar
		}
	}
	return res
}

func (m *Matrix) Mul(other *Matrix) (*Matrix, error) {
	if m.Cols != other.Rows {
		return nil, fmt.Errorf("cannot multiply matrices of shape (%d, %d) and (%d, %d)", m.Rows, m.Cols, other.Rows, other.Cols)
	}
	res := NewMatrix(m.Rows, other.Cols)
	for i := 0; i < m.Rows; i++ {
		for j := 0; j < other.Cols; j++ {
			var sum float64
			for k := 0; k < m.Cols; k++ {
				sum += m.Data[i][k] * other.Data[k][j]
			}
			res.Data[i][j] = sum
		}
	}
	return res, nil
}

func (m *Matrix) MulVec(v *Vector) (*Vector, error) {
	if m.Cols != len(v.Elements) {
		return nil, fmt.Errorf("cannot multiply matrix of shape (%d, %d) with vector of dim %d", m.Rows, m.Cols, len(v.Elements))
	}
	res := make([]float64, m.Rows)
	for i := 0; i < m.Rows; i++ {
		var sum float64
		for j := 0; j < m.Cols; j++ {
			sum += m.Data[i][j] * v.Elements[j]
		}
		res[i] = sum
	}
	return &Vector{Elements: res}, nil
}

func (m *Matrix) Transpose() *Matrix {
	res := NewMatrix(m.Cols, m.Rows)
	for i := 0; i < m.Rows; i++ {
		for j := 0; j < m.Cols; j++ {
			res.Data[j][i] = m.Data[i][j]
		}
	}
	return res
}

func (m *Matrix) Det() (float64, error) {
	if m.Rows != m.Cols {
		return 0, fmt.Errorf("determinant is only defined for square matrices (got %d x %d)", m.Rows, m.Cols)
	}
	n := m.Rows
	if n == 1 {
		return m.Data[0][0], nil
	}
	if n == 2 {
		return m.Data[0][0]*m.Data[1][1] - m.Data[0][1]*m.Data[1][0], nil
	}

	// Gaussian elimination with partial pivoting
	a := make([][]float64, n)
	for i := range a {
		a[i] = make([]float64, n)
		copy(a[i], m.Data[i])
	}

	det := 1.0
	for i := 0; i < n; i++ {
		// Find pivot
		pivotRow := i
		for k := i + 1; k < n; k++ {
			if math.Abs(a[k][i]) > math.Abs(a[pivotRow][i]) {
				pivotRow = k
			}
		}

		if math.Abs(a[pivotRow][i]) < 1e-12 {
			return 0, nil
		}

		if pivotRow != i {
			a[i], a[pivotRow] = a[pivotRow], a[i]
			det = -det
		}

		pivot := a[i][i]
		det *= pivot

		for k := i + 1; k < n; k++ {
			factor := a[k][i] / pivot
			for j := i; j < n; j++ {
				a[k][j] -= factor * a[i][j]
			}
		}
	}

	return det, nil
}

func (m *Matrix) Inv() (*Matrix, error) {
	if m.Rows != m.Cols {
		return nil, fmt.Errorf("inverse is only defined for square matrices (got %d x %d)", m.Rows, m.Cols)
	}
	n := m.Rows

	// Augmented matrix [A | I]
	aug := make([][]float64, n)
	for i := 0; i < n; i++ {
		aug[i] = make([]float64, 2*n)
		copy(aug[i][:n], m.Data[i])
		aug[i][n+i] = 1.0
	}

	// Gauss-Jordan elimination
	for i := 0; i < n; i++ {
		pivotRow := i
		for k := i + 1; k < n; k++ {
			if math.Abs(aug[k][i]) > math.Abs(aug[pivotRow][i]) {
				pivotRow = k
			}
		}

		if math.Abs(aug[pivotRow][i]) < 1e-12 {
			return nil, fmt.Errorf("matrix is singular and cannot be inverted")
		}

		if pivotRow != i {
			aug[i], aug[pivotRow] = aug[pivotRow], aug[i]
		}

		pivot := aug[i][i]
		for j := 0; j < 2*n; j++ {
			aug[i][j] /= pivot
		}

		for k := 0; k < n; k++ {
			if k != i {
				factor := aug[k][i]
				for j := 0; j < 2*n; j++ {
					aug[k][j] -= factor * aug[i][j]
				}
			}
		}
	}

	inv := NewMatrix(n, n)
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			inv.Data[i][j] = aug[i][n+j]
		}
	}
	return inv, nil
}

func (m *Matrix) ToArray() *Array {
	rows := make([]Object, m.Rows)
	for i := 0; i < m.Rows; i++ {
		cols := make([]Object, m.Cols)
		for j := 0; j < m.Cols; j++ {
			cols[j] = &Float{Value: m.Data[i][j]}
		}
		rows[i] = &Array{Elements: cols}
	}
	return &Array{Elements: rows}
}
