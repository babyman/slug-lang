# Slug Documentation Site

This directory contains the source code for the [sluglang.org](https://sluglang.org) documentation site, built
using [Jekyll](https://jekyllrb.com/) and the [Chulapa theme](https://dieghernan.github.io/chulapa/).

## 🚀 Quick Start

### Prerequisites

You will need **Ruby** (v3.0 or higher) and **Bundler** installed on your system.

### 1. Install Dependencies

From this directory, run:

```bash
bundle install
```

### 2. Run Locally
Start the Jekyll server to preview your changes:
```shell script
bundle exec jekyll serve
```

Once started, the site will be available at `http://localhost:4000`. The server will automatically rebuild the site when
you save changes to files.

## 📂 Project Structure

- `_adrs/`: Architecture Decision Records (Collection).
- `_setup/`: Content fragments for the installation guide.
- `_developers-guide/`: Detailed technical documentation.
- `assets/`: Images, custom CSS, and site icons.
- `_config.yml`: The main configuration file (navigation, theme settings, and plugins).
- `index.md`: The site landing page.

## ✍️ Content Management

### Adding a new ADR

1. Create a new `.md` file in `_adrs/`.
2. Follow the naming convention `ADR-XXX.md`.
3. Ensure you include the required front matter:

```yaml
---
title: "Your Title"
date: YYYY-MM-DD
---
```

### Navigation

The top navigation bar is controlled via the `navbar` section in `_config.yml`.

### Architecture Note

This site uses **Collections** for ADRs and Setup guides.

- Individual items live in folders starting with an underscore (e.g., `_adrs/`).
- The "Archive" or listing pages (e.g., `adr.md`, `setup.md`) are located in the root of the `docs` folder to provide
  clean URLs like `/adr/`.

## 🚢 Deployment

The site is configured for **GitHub Pages**. Simply push changes to the `master` (or `main`) branch, and GitHub will
automatically trigger a build using the `Gemfile` provided.
