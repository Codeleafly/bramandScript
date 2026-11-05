# Scientific Computing & Linear Algebra in BramandScript

This example demonstrates how to perform high-performance linear algebra and matrix operations using `std/math/tensor`.

## Features Demonstrated

1. **Multi-Dimensional Tensors (`std/math/tensor.bmd`)**:
   - Arbitrary row-major and strided representations.
   - Coordinate indexing (`tensor.get([row, col])`).
   - Matrix Transpose ($A^T$).
   - Matrix Multiplication ($A \cdot B$) with dimension compatibility checks.

2. **Gaussian Inversion & Determinants**:
   - Recursive/LU decomposition determinant calculation ($\det(A)$).
   - Gauss-Jordan elimination matrix inversion ($A^{-1}$).
   - Identity verification ($A \cdot A^{-1} \approx I$).

3. **Solving Linear Systems**:
   - Solves systems of equations of the form $A \cdot x = b$ using matrix inversion:
     $$x = A^{-1} \cdot b$$

4. **Statistical Reductions**:
   - `sum()`, `mean()`, `max()`, `min()`.

## Running the Example

```bash
bramand run examples/11_tensor_linear_algebra/main.bmd
```
