#!/usr/bin/env bash
set -euo pipefail

project_dir="$(cd "$(dirname "$0")/.." && pwd)"
cd "$project_dir"
mkdir -p bin dist/examples dist/skill/pointfold
go build -o bin/pointfold .
cp index.html dist/index.html
cp FORMAT.md dist/FORMAT.md
cp skill/pointfold/SKILL.md dist/skill/pointfold/SKILL.md
cp -R examples/assets dist/examples/
for source in examples/*.md; do
  name="$(basename "$source" .md)"
  bin/pointfold render "$source" -o "dist/examples/$name.html"
  cp "$source" "dist/examples/$name.md"
done
touch dist/.nojekyll
