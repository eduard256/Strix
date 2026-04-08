---
name: frontend_design_strix
description: Create or redesign frontend pages for Strix. Use when building new HTML pages, redesigning existing ones, or working on any UI task in the www/ directory. Covers design principles, layout patterns, and component usage.
disable-model-invocation: true
---

# Strix Frontend Design

You are creating or modifying a frontend page for Strix. Your goal is to produce a page that looks **identical in quality** to the existing pages, especially `www/homekit.html` which is the design reference.

## Before you start

Read these files completely:

1. **`www/design-system.html`** -- All CSS variables, every component, JS patterns. This is your component library.
2. **`www/homekit.html`** -- The design reference. This page is the gold standard. Study its structure, spacing, how little text it uses, how the back button is positioned.
3. **`www/index.html`** -- The entry point. Understand the probe flow and how data is passed between pages via URL params.

If you need to understand backend APIs or the probe system, read:
- `www/standard.html` -- how probe data flows into a configuration page
- `www/test.html` -- how polling and real-time updates work
- `www/config.html` -- complex two-column layout with live preview

## Design Philosophy

### Radical minimalism

Every element on screen must earn its place. If something doesn't help the user complete their task, remove it.

- **10% text, 90% meaning.** A label that says "Pairing Code" with an info-icon is better than a paragraph explaining what a pairing code is.
- **Hide details behind info-icons.** Long explanations go into tooltips (the `(i)` icon pattern). The user who needs the explanation can hover. The user who doesn't is not bothered.
- **No decorative elements without function.** No ornamental icons, no badges that don't convey information, no cards-as-decoration.
- **One action per screen.** Each page should have one primary thing the user does. Everything else is secondary.

### How we think about design decisions

When building homekit.html, we went through this process:

1. **Started with all the data** -- device info table, long descriptions, badges, decorative icons
2. **Asked "does the user need this?"** for every element
3. **Removed everything that wasn't essential** -- the device info table (IP, MAC, vendor) was removed because the user doesn't need it to enter a PIN code
4. **Moved explanations into tooltips** -- "This camera supports Apple HomeKit. Enter the 8-digit pairing code printed on your camera or included in the manual" became just a label "Pairing Code" with a tooltip
5. **Removed format hints** -- "Format: XXX-XX-XXX" was removed because the input fields themselves make the format obvious
6. **Made the primary action obvious** -- big button, full width, impossible to miss

Apply this same thinking to every page you create.

### Visual rules

- Dark theme with purple accent -- never deviate from the color palette in `:root`
- All icons are inline SVG -- never use emoji, never use icon fonts, never use external icon libraries
- Fonts: system font stack for UI, monospace for technical values (URLs, IPs, codes)
- Borders are subtle: `rgba(139, 92, 246, 0.15)` -- barely visible purple tint
- Glow effects on focus and hover, never on static elements (except logos)
- Animations are fast (150ms) and subtle -- translateY(-2px) on hover, fadeIn on page load
- No rounded corners larger than 8px (except special cases like toggle switches)

## Layout Patterns

### Pages after probe (like homekit.html) -- TRUE CENTER

This is the most common case for new pages. Content is vertically centered on screen.

```
.screen {
    min-height: 100vh;
    display: flex;
    align-items: center;    /* TRUE CENTER -- not flex-start */
    justify-content: center;
}
.container { max-width: 480px; width: 100%; }
```

**Back button** is positioned OUTSIDE the container, wider than content, using `.back-wrapper`:

```
.back-wrapper {
    position: absolute; top: 1.5rem;
    left: 50%; transform: translateX(-50%);
    width: 100%; max-width: 600px;    /* wider than container */
    padding: 0 1.5rem;
    z-index: 10;
}
```

This is MANDATORY for all centered layout pages. The back button must NOT be inside the centered container.

### Entry page (like index.html) -- TOP CENTER

Content is near the top with `margin-top: 8vh`. Used for the main entry point only.

### Content pages (like standard.html, create.html) -- STANDARD

Back button at top, then title, then content flowing down. `max-width: 600px`, no vertical centering.

### Data-heavy pages (like test.html) -- WIDE

`max-width: 1200px` with card grids.

### Two-column (like config.html) -- SPLIT

Settings left, live preview right. Collapses to tabs on mobile.

## Hero Section Pattern

For centered pages, the hero contains a logo/icon + short title:

```html
<div class="hero">
    <svg class="logo-icon">...</svg>    <!-- 48-72px, with glow filter -->
    <h1 class="title">Short Name</h1>   <!-- 1.25rem, white, font-weight 600 -->
</div>
```

- The icon should be recognizable and relevant (Strix owl for main, HomeKit house for HomeKit)
- The title is SHORT -- one or two words max
- No subtitles unless absolutely necessary
- Glow effect on the icon via `filter: drop-shadow()`

## Component Usage

All components are documented with live examples in `www/design-system.html`. Key ones:

- **Buttons**: `.btn .btn-primary .btn-large` for primary action (full width), `.btn-outline` for secondary
- **Inputs**: `.input` with `.label` and optional `.info-icon` with `.tooltip`
- **Toast**: Every page needs `<div id="toast" class="toast hidden"></div>` and the `showToast()` function
- **Error box**: `.error-box` with `.visible` class toggled
- **Info icon + tooltip**: For hiding explanations -- always prefer this over visible text

## Navigation -- CRITICAL

### Always pass ALL known data forward

When navigating to another page, pass every piece of data you have. This is non-negotiable. Future pages may need any of these values.

```javascript
function navigateNext() {
    var p = new URLSearchParams();
    p.set('primary_data', value);
    // Pass through EVERYTHING known:
    if (ip) p.set('ip', ip);
    if (mac) p.set('mac', mac);
    if (vendor) p.set('vendor', vendor);
    if (model) p.set('model', model);
    if (server) p.set('server', server);
    if (hostname) p.set('hostname', hostname);
    if (ports) p.set('ports', ports);
    if (user) p.set('user', user);
    if (channel) p.set('channel', channel);
    // ... any other params from probe
    window.location.href = 'next.html?' + p.toString();
}
```

### Page init always reads all params

```javascript
var params = new URLSearchParams(location.search);
var ip = params.get('ip') || '';
var mac = params.get('mac') || '';
var vendor = params.get('vendor') || '';
// ... read ALL possible params even if this page doesn't use them
// They need to be available for passing to the next page
```

## JavaScript Rules

- Use `var`, not `let`/`const` -- ES5 compatible
- Build DOM with `document.createElement`, not innerHTML
- Use `async function` + `fetch()` for API calls
- Always handle errors: check `!r.ok`, catch exceptions, show toast
- Debounce input handlers if they trigger API calls (300ms)
- Use `addEventListener`, never inline event handlers in HTML

## API Pattern

```javascript
async function doSomething() {
    try {
        var r = await fetch('api/endpoint', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(payload)
        });
        if (!r.ok) {
            var text = await r.text();
            showToast(text || 'Error ' + r.status);
            return;
        }
        var data = await r.json();
        // success...
    } catch (e) {
        showToast('Connection error: ' + e.message);
    }
}
```

## Checklist before finishing

- [ ] Page uses correct layout pattern for its type
- [ ] Back button positioned correctly (`.back-wrapper` for centered, inline for standard)
- [ ] All CSS variables from `:root` -- no hardcoded colors
- [ ] No unnecessary text -- everything possible hidden behind info-icons
- [ ] All known URL params are read at init and passed forward on navigation
- [ ] Toast element present, showToast function included
- [ ] Error states handled (API errors, validation)
- [ ] Mobile responsive (test at 375px width)
- [ ] No emoji anywhere
- [ ] All icons are inline SVG
- [ ] Primary action is obvious and full-width
- [ ] Page looks like it belongs with homekit.html and index.html
