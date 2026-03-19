module.exports = {
  "frontend/**/*.{js,ts,html,css,scss}": [
    "npx run lint --prefix frontend",
    "cd frontend && npx prettier --write --plugin=prettier-plugin-tailwindcss"
  ],
  "backend/**/*.go": [
    "cd backend && gofmt -w",
    "cd backend && golangci-lint run"
  ]
}