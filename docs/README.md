# GoRetro Documentation Site

This directory contains the GitHub Pages documentation website for GoRetro.

## Viewing the Documentation

Once GitHub Pages is enabled, the documentation will be available at:
`https://armatorix.github.io/GoRetro/`

## Enabling GitHub Pages

To enable GitHub Pages for this repository:

1. Go to your repository on GitHub
2. Click on **Settings**
3. Scroll down to the **Pages** section in the left sidebar
4. Under **Source**, select:
   - **Branch:** `main` (or your default branch)
   - **Folder:** `/docs`
5. Click **Save**
6. Wait a few minutes for the site to build and deploy
7. Your site will be available at `https://armatorix.github.io/GoRetro/`

## Local Development

To preview the documentation locally:

```bash
# Using Python's built-in HTTP server
cd docs
python3 -m http.server 8000

# Then open http://localhost:8000 in your browser
```

Or use any other static file server of your choice.

## Site Structure

- `index.html` - Main documentation page with tabbed interface
  - **Features Tab** - Core features, phases, and AI capabilities
  - **Configuration Tab** - Environment variables and configuration examples
  - **Deployment Tab** - Docker Compose and standalone deployment guides
  - **Usage Guide Tab** - Step-by-step usage instructions and best practices

## Customization

To customize the documentation:

1. Edit `index.html` directly
2. The site uses inline CSS for easy maintenance
3. All content is in a single HTML file for simplicity
4. Colors and styling can be adjusted in the `:root` CSS variables

## Features

- 📱 Responsive design (mobile-friendly)
- 🎨 Modern, clean interface with gradient headers
- 📑 Tabbed navigation for easy content organization
- 🔗 Deep linking support (e.g., `#configuration` jumps to that tab)
- ♿ Accessible color contrasts and semantic HTML
- ⚡ Fast loading (single HTML file, no external dependencies)
