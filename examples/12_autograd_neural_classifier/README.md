# Neural Network Training & Autograd Engine in BramandScript

This example demonstrates how to build and train artificial neural networks from first principles using BramandScript's reverse-mode automatic differentiation engine (`std/ml`).

## Features Demonstrated

1. **Scalar Autograd Engine (`std/ml/autograd.bmd`)**:
   - Dynamic Directed Acyclic Graph (DAG) construction during the forward pass.
   - Exact backpropagation via topological sort (`backward()`).
   - Scalar mathematical operations: `add`, `mul`, `sub`, `relu`, `tanh_val`, `pow_val`.

2. **Modular Architecture**:
   - `Neuron`: Linear combinations of weights and inputs with activation functions.
   - `Layer`: Dense collection of parallel neurons.
   - `MLP`: Multi-Layer Perceptron pipeline with stacked hidden layers.

3. **Gradient Descent Optimizer**:
   - Loss computation (Mean Squared Error).
   - Zeroing gradients (`model.zero_grad()`).
   - Weight updates in direction of negative gradient (`model.step(learning_rate)`).

## Running the Example

```bash
bramand run examples/12_autograd_neural_classifier/main.bmd
```
