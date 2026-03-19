module.exports = {
  "frontend/**/*.{js,ts,html,css,scss}": [
    "npx run lint --prefix frontend",
    "npx prettier --write"
  ],
  "backend/**/*.go": [
    "gofmt -w",
    "golangci-lint run"
  ]
}