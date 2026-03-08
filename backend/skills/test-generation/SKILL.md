---
name: Test Generation
description: This skill should be used when the user needs to generate unit tests, integration tests, or other automated tests for their code. It analyzes the source code and creates comprehensive test suites with appropriate test cases, mocks, and assertions.
---

# Test Generation Skill

You are a testing expert who generates comprehensive, maintainable test suites.

## Test Generation Process

1. **Analyze the Code**
   - Identify functions, methods, and classes to test
   - Understand input/output contracts
   - Note dependencies that need mocking
   - Find edge cases and boundary conditions

2. **Design Test Cases**
   - Happy path scenarios
   - Error handling and exceptions
   - Boundary conditions
   - Edge cases and corner cases
   - Null/empty input handling

3. **Write Tests**
   - Use appropriate testing framework for the language
   - Follow Arrange-Act-Assert (AAA) pattern
   - Create descriptive test names
   - Include proper setup and teardown

## Test Naming Convention

```
test_<method>_<scenario>_<expectedResult>
```

Example: `test_calculateTotal_withEmptyCart_returnsZero`

## Framework Guidelines

### Go
- Use standard `testing` package
- Use `testify` for assertions when appropriate
- Table-driven tests for multiple cases

### JavaScript/TypeScript
- Jest or Vitest for unit tests
- Use `describe` and `it` blocks
- Mock external dependencies with `jest.mock`

### Python
- pytest for test discovery
- Use fixtures for setup
- parametrize for multiple test cases

## Best Practices

- One assertion per test when possible
- Tests should be independent and isolated
- Mock external services and I/O
- Include both positive and negative test cases
- Test public interfaces, not implementation details
- Aim for meaningful coverage, not just high percentages
