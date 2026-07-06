#!/usr/bin/env python3
"""
Static documentation site generator for GitHub Pages.

Renders every Markdown file in docs/ (plus the repo README) into a styled,
self-contained HTML site under _site/. No external CSS/JS/CDN — everything is
inlined so it works offline and in restricted previews.

Run:  python docs/_build.py
"""
import os
import re
import shutil
import markdown

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
DOCS = os.path.join(ROOT, "docs")
OUT = os.path.join(ROOT, "_site")

# ── Project detection (koris vs knode) ──────────────────────────────────────
def detect_project():
    gomod = os.path.join(ROOT, "go.mod")
    name = "Koris"
    if os.path.exists(gomod):
        head = open(gomod).readline()
        if "knode" in head:
            name = "knode"
        elif "koris" in head:
            name = "Koris"
    # fall back to folder name
    return name

PROJECT = detect_project()
EMOJI = "🛰️" if PROJECT == "knode" else "🛡️"
TAGLINE = (
    "VPN node agent for the Koris platform"
    if PROJECT == "knode"
    else "Multi-protocol VPN management platform"
)

# ── Nav ordering: nice titles for known pages ───────────────────────────────
ORDER = [
    ("index.html", "🏠 Home"),
    ("installation.html", "📘 Installation"),
    ("architecture.html", "🏛️ Architecture"),
    ("configuration.html", "⚙️ Configuration"),
    ("protocols.html", "🔌 Protocols"),
    ("nodes.html", "🛰️ Nodes"),
    ("api.html", "📡 API"),
    ("ui-ux.html", "🎨 UI / UX"),
    ("ADMIN.html", "🛠️ Admin Guide"),
    ("API.html", "📡 HTTP API"),
    ("DOCKER.html", "🐳 Docker"),
    ("low-memory-tuning.html", "🧠 Low-memory Tuning"),
]

MD_EXT = [
    "extra", "toc", "sane_lists", "admonition",
    "pymdownx.superfences", "pymdownx.tabbed", "pymdownx.tilde",
    "tables", "fenced_code", "codehilite",
]

CSS = """
:root{--bg:#070a12;--surface:#0b1120;--surface2:#131c2e;--border:#28333f;
--text:#e6edf3;--muted:#8b98a5;--primary:#22d3ee;--brand:#7c5cff;
--radius:12px;--mono:'SFMono-Regular',Consolas,Menlo,monospace}
*{box-sizing:border-box}
html{scroll-behavior:smooth}
body{margin:0;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;
background:radial-gradient(1200px 600px at 100% -10%,rgba(124,92,255,.10),transparent 60%),
radial-gradient(1000px 500px at -10% 110%,rgba(34,211,238,.08),transparent 60%),var(--bg);
color:var(--text);line-height:1.65}
a{color:var(--primary);text-decoration:none}
a:hover{text-decoration:underline}
.layout{display:flex;min-height:100vh}
.sidebar{width:260px;flex:0 0 260px;background:rgba(11,17,32,.7);backdrop-filter:blur(10px);
border-right:1px solid var(--border);padding:24px 16px;position:sticky;top:0;height:100vh;overflow-y:auto}
.brand{display:flex;align-items:center;gap:10px;font-weight:800;font-size:20px;margin-bottom:4px}
.brand .em{font-size:24px}
.tagline{color:var(--muted);font-size:12.5px;margin-bottom:20px}
.nav a{display:block;padding:8px 12px;border-radius:8px;color:var(--text);font-size:14px;margin:2px 0;
transition:background .15s,color .15s}
.nav a:hover{background:var(--surface2);text-decoration:none}
.nav a.active{background:linear-gradient(135deg,rgba(34,211,238,.15),rgba(124,92,255,.15));
color:var(--primary);font-weight:600}
.content{flex:1;max-width:900px;margin:0 auto;padding:48px 40px 96px}
.content h1{font-size:34px;letter-spacing:-.02em;margin-top:0}
.content h2{margin-top:40px;padding-bottom:8px;border-bottom:1px solid var(--border)}
.content h3{margin-top:28px}
.content code{font-family:var(--mono);font-size:.9em;background:var(--surface2);
padding:2px 6px;border-radius:6px}
.content pre{background:#05070d;border:1px solid var(--border);border-radius:var(--radius);
padding:16px 18px;overflow-x:auto}
.content pre code{background:none;padding:0}
.content table{border-collapse:collapse;width:100%;margin:16px 0;font-size:14px}
.content th,.content td{border:1px solid var(--border);padding:8px 12px;text-align:left}
.content th{background:var(--surface2);text-transform:uppercase;font-size:12px;letter-spacing:.5px;color:var(--muted)}
.content tr:hover td{background:rgba(255,255,255,.02)}
.content blockquote{border-left:3px solid var(--primary);margin:16px 0;padding:4px 18px;
background:var(--surface);border-radius:0 8px 8px 0;color:#cdd6df}
.content img{max-width:100%}
.footer{margin-top:64px;padding-top:24px;border-top:1px solid var(--border);color:var(--muted);font-size:13px}
::-webkit-scrollbar{width:10px}::-webkit-scrollbar-thumb{background:rgba(139,152,165,.4);border-radius:9999px}
.menu-btn{display:none}
@media(max-width:820px){
 .sidebar{position:fixed;left:-280px;z-index:50;transition:left .2s}
 .sidebar.open{left:0}
 .content{padding:64px 20px 80px}
 .menu-btn{display:block;position:fixed;top:14px;left:14px;z-index:60;background:var(--surface2);
  border:1px solid var(--border);color:var(--text);border-radius:8px;padding:8px 12px;font-size:16px;cursor:pointer}
}
"""

JS = """
<script>
const btn=document.querySelector('.menu-btn');
if(btn){btn.addEventListener('click',()=>document.querySelector('.sidebar').classList.toggle('open'))}
</script>
"""

def slug_to_html(fname):
    return re.sub(r"\.md$", ".html", fname)

def title_for(html_name, md_text):
    for h, t in ORDER:
        if h == html_name:
            return t
    m = re.search(r"^#\s+(.+)$", md_text, re.M)
    return (m.group(1).strip() if m else html_name.replace(".html", "").title())

def build_nav(pages, current):
    known = [h for h, _ in ORDER]
    items = []
    for h, label in ORDER:
        if h in pages:
            items.append((h, label))
    for h in sorted(pages):
        if h not in known:
            items.append((h, pages[h]["nav"]))
    out = ['<nav class="nav">']
    for h, label in items:
        cls = ' class="active"' if h == current else ""
        out.append(f'<a href="{h}"{cls}>{label}</a>')
    out.append("</nav>")
    return "\n".join(out)

def page_html(title, body, nav):
    return f"""<!doctype html><html lang="en"><head>
<meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>{title} · {PROJECT} docs</title><style>{CSS}</style></head><body>
<button class="menu-btn">☰</button>
<div class="layout">
<aside class="sidebar">
<div class="brand"><span class="em">{EMOJI}</span> {PROJECT}</div>
<div class="tagline">{TAGLINE}</div>
{nav}
</aside>
<main class="content">
{body}
<div class="footer">📖 {PROJECT} documentation · built for GitHub Pages ·
<a href="https://github.com/anonysec/{PROJECT.lower()}">GitHub</a></div>
</main></div>{JS}</body></html>"""

def fix_links(html, page_names):
    """Rewrite intra-doc .md links to .html. Links whose target isn't part of
    the generated site (e.g. SECURITY.md, CONTRIBUTING.md, ../internal/...) are
    pointed at the GitHub repo instead so they never 404."""
    repo = f"https://github.com/anonysec/{PROJECT.lower()}/blob/HEAD/"

    def repl(m):
        target, anchor = m.group(1), (m.group(2) or "")
        base = target.split("/")[-1]  # strip docs/ or ../ prefixes
        if base.lower() in ("readme",):
            name = "index.html"
        else:
            name = base + ".html"
        if name in page_names:
            return f'href="{name}{anchor}"'
        # not a generated page → link to the file on GitHub
        clean = target.lstrip("./")
        return f'href="{repo}{clean}.md{anchor}"'

    return re.sub(r'href="([^":]+?)\.md(#[^"]*)?"', repl, html)

def main():
    if os.path.exists(OUT):
        shutil.rmtree(OUT)
    os.makedirs(OUT)

    md = markdown.Markdown(extensions=MD_EXT)

    # Collect sources: docs/*.md (skip files starting with _) + README as index
    sources = {}
    for f in os.listdir(DOCS):
        if f.endswith(".md") and not f.startswith("_"):
            html_name = slug_to_html(f)
            if f.lower() == "readme.md":
                html_name = "index.html"
            sources[html_name] = os.path.join(DOCS, f)

    # README.md from repo root → home (overrides docs/README if both)
    root_readme = os.path.join(ROOT, "README.md")
    if os.path.exists(root_readme):
        sources["index.html"] = root_readme

    # first pass: titles for nav
    pages = {}
    for html_name, path in sources.items():
        text = open(path, encoding="utf-8").read()
        pages[html_name] = {"path": path, "text": text,
                            "nav": title_for(html_name, text)}

    # render
    for html_name, meta in pages.items():
        md.reset()
        body = md.convert(meta["text"])
        body = fix_links(body, set(pages.keys()))
        nav = build_nav(pages, html_name)
        title = "Home" if html_name == "index.html" else meta["nav"]
        html = page_html(title, body, nav)
        with open(os.path.join(OUT, html_name), "w", encoding="utf-8") as fh:
            fh.write(html)

    # .nojekyll so GitHub serves our files as-is
    open(os.path.join(OUT, ".nojekyll"), "w").close()
    print(f"Built {len(pages)} pages → {OUT}")
    for p in sorted(pages):
        print("  •", p)

if __name__ == "__main__":
    main()
