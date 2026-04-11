# go-edit

Code editor widget for [go-gui](https://github.com/mike-ward/go-gui). Pure
Go, no CGO. Syntax highlighting via
[chroma](https://github.com/alecthomas/chroma). Text shaping via
[go-glyph](https://github.com/mike-ward/go-glyph).

Status: Phase 0 skeleton. See [ROADMAP.md](ROADMAP.md).

## Development

Local dev against sibling checkouts of `go-gui` and `go-glyph`: add
`replace` directives to `go.mod` pointing at `../go-gui` and `../go-glyph`.
Strip before tagging.

## License

MIT. See [LICENSE](LICENSE).
