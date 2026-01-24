# Development Tools

## Agentation

Agentation is a visual feedback tool for AI coding agents that's enabled in development mode only.

### What is Agentation?

Agentation (agent + annotation) allows you to annotate elements on your webpage and generate structured feedback for AI coding agents like Claude Code, Cursor, or Windsurf. It captures class names, selectors, and positions so agents can locate the exact source files that need changes.

### How to Use

1. **Activate**: Look for the Agentation icon in the bottom-right corner of your webapp (in dev mode)
2. **Hover**: Move your mouse over elements to see their names highlighted
3. **Click**: Click any element to add an annotation
4. **Write Feedback**: Add your notes about what needs to be changed or fixed
5. **Copy**: Click the copy icon to get formatted markdown output
6. **Paste**: Paste the output into your AI agent (Cursor, Claude Code, etc.)

### Features

- **Element Selection**: Click any UI element to annotate it
- **Text Selection**: Select specific text for typos or content issues
- **Multi-select**: Annotate multiple elements at once
- **Animation Pause**: Freeze CSS animations to annotate specific states
- **Layout Feedback**: Click empty areas to provide layout feedback

### Best Practices

- Be specific with your feedback ("Button text should be 'Submit' not 'Send'")
- Create one issue per annotation for easier agent processing
- Include context about expected vs. actual behavior
- Use text selection for precise content issues
- Pause animations when annotating animated elements

### When It's Active

Agentation only loads in development mode (`npm run dev`) and will not be included in production builds. It has zero runtime impact on your production application.

### Integration

The Agentation component is automatically loaded in the App component and requires no additional configuration. It will only initialize when `import.meta.env.DEV` is true.
