// Draws stability the way the design package does (design:
// src/components/stability/ and src/components/maturity/), each with a
// popover (hover, focus, or tap) saying what it means:
//
//   - Scalar's stability badge on each operation becomes a StabilityMark: the
//     Track and the level's name, in the level's color. Scalar draws an
//     x-badges badge as a plain pill; the docs bundler ends each stability
//     badge's name with STABILITY_MARK, so this finds exactly those, and
//     leaves Scalar's own pill in place if it cannot.
//   - Each operation in the navigation gets a short StabilityMark, the Track
//     alone, and each tag a MaturityRing worked out from its operations'
//     levels. The docs page puts every level in window.EXALYNT_STABILITY,
//     since the navigation only draws a tag's operations once it's opened.
//
// The levels and maturity's rules mirror the design package's
// stability/stability.ts and maturity/maturity.ts; change them there first.
(() => {
  // U+2060 WORD JOINER: invisible, so a badge reads normally if this script
  // never runs. Keep in step with stabilityMark in internal/handlers.
  const STABILITY_MARK = "⁠";

  const STABILITY_DOCS_URL = "https://readme.exalynt.com/how-it-works/stability-levels";
  const MATURITY_DOCS_URL = "https://readme.exalynt.com/how-it-works/maturity";

  const { operations = {}, tags = {} } = window.EXALYNT_STABILITY || {};

  // Each level, and the maturity band at its top, whose color the band takes.
  const LEVELS = [
    {
      id: "prototype",
      name: "Prototype",
      fullName: "Prototype",
      stability: "Exploring",
      summary:
        "Explores an idea so we can choose a direction. Its data may be fake, and it may change completely or be thrown away.",
      colors: { light: "#6d6a7c", dark: "#aeabbe" },
      band: "Experimental",
      bandSummary:
        "Early work. On average, its endpoints are still being explored, so expect much of it to change, and some of its data may be fake.",
    },
    {
      id: "alpha",
      name: "Alpha",
      fullName: "Alpha",
      stability: "Developing",
      summary: "Works, but is still taking shape. Expect frequent breaking changes and bugs.",
      colors: { light: "#9c4f6c", dark: "#de9cb6" },
      band: "Emerging",
      bandSummary:
        "Taking shape. On average, its endpoints work but are still changing often, and bugs are to be expected.",
    },
    {
      id: "beta",
      name: "Beta",
      fullName: "Beta",
      stability: "Refining",
      summary:
        "Settled enough for real use. Breaking changes and bugs are occasional. We aim to announce breaking changes ahead of time, but can't always.",
      colors: { light: "#3f6f88", dark: "#8ab4cc" },
      band: "Promising",
      bandSummary:
        "Settled enough for real use. On average, its endpoints are being refined, with occasional breaking changes we aim to announce ahead of time.",
    },
    {
      id: "ga",
      name: "GA",
      fullName: "General Availability",
      stability: "Stable",
      summary:
        "Generally available and stable. It keeps working as it does today; breaking changes only come in a new version.",
      colors: { light: "#4c7a58", dark: "#94bd9f" },
      band: "Established",
      bandSummary: "Stable and dependable. On average, its endpoints keep working as they do today.",
    },
  ];

  const STABILITY_ABOUT =
    "Stability is how much something will still change, and how far you can rely on it today. It moves up from Prototype through Alpha and Beta to General Availability as it’s proven in real use.";

  const MATURITY_ABOUT =
    "Maturity is how complete and stable a group of endpoints is, as presently known, from each endpoint’s stability. It can go down as well as up as endpoints are added.";

  // The level's position, 1 (Prototype) to 4 (GA), or 0 if it's unknown.
  const stepOf = (id) => LEVELS.findIndex((l) => l.id === id) + 1;

  // A tag's maturity: each operation counts 25% for every level it has
  // reached, averaged and rounded.
  function maturity(levels) {
    const total = levels.reduce((sum, id) => sum + stepOf(id) * 25, 0);
    return Math.round(total / levels.length);
  }

  // The level whose quarter a maturity is in: 0–24% is Prototype's, 25–49%
  // Alpha's, 50–74% Beta's, and 75–100% GA's.
  const bandOf = (progress) => LEVELS[Math.min(Math.floor(progress / 25), 3)];

  function el(tag, className, ...children) {
    const node = document.createElement(tag);
    if (className) node.className = className;
    node.append(...children);
    return node;
  }

  const SVG = "http://www.w3.org/2000/svg";

  function svg(tag, attributes) {
    const node = document.createElementNS(SVG, tag);
    for (const [name, value] of Object.entries(attributes)) node.setAttribute(name, value);
    return node;
  }

  function setColors(node, level) {
    node.style.setProperty("--stability-light", level.colors.light);
    node.style.setProperty("--stability-dark", level.colors.dark);
  }

  // cells returns one element per level, lit up to step.
  function cells(className, step) {
    return LEVELS.map((_, i) => el("span", i < step ? `${className} lit` : className));
  }

  // mark draws a StabilityMark: long, with the level's name, or short, the
  // Track alone, named for assistive tech. `feature` is what it's on, for
  // the popover to say "Make an estimate is in Alpha."
  function mark(level, { short = false, feature = "" } = {}) {
    const track = el("span", "stability-track", ...cells("stability-cell", stepOf(level.id)));
    track.setAttribute("aria-hidden", "true");
    const node = el("span", "stability-mark", track);
    if (short) {
      node.setAttribute("role", "img");
      node.setAttribute("aria-label", `${level.name} stability`);
    } else {
      node.append(level.name);
    }
    node.dataset.stability = level.id;
    if (feature) node.dataset.feature = feature;
    setColors(node, level);
    return node;
  }

  // ringGlyph draws maturity's one look: a ring in four quarter arcs,
  // clockwise from the top (Prototype, Alpha, Beta, GA), each filled as far
  // as the maturity reaches into it, in the band's color, with the
  // percentage inside from 40px across, and beside it below that.
  function ringGlyph(progress, size) {
    const stroke = Math.max(3, Math.round(size / 11));
    const radius = (size - stroke) / 2;
    const circumference = 2 * Math.PI * radius;
    const quarter = circumference / 4;
    const gap = stroke * 0.9;
    const arc = quarter - gap;
    const inside = size >= 40;

    const ring = svg("svg", { viewBox: `0 0 ${size} ${size}`, width: size, height: size });
    LEVELS.forEach((_, index) => {
      const fill = Math.min(Math.max((progress - index * 25) / 25, 0), 1);
      const common = {
        cx: size / 2,
        cy: size / 2,
        r: radius,
        fill: "none",
        "stroke-width": stroke,
        "stroke-dashoffset": -(index * quarter + gap / 2),
      };
      ring.append(
        svg("circle", { ...common, class: "maturity-track", "stroke-dasharray": `${arc} ${circumference}` }),
      );
      if (fill > 0) {
        ring.append(
          svg("circle", {
            ...common,
            stroke: "currentColor",
            "stroke-dasharray": `${arc * fill} ${circumference}`,
          }),
        );
      }
    });

    const figure = el("span", "maturity-figure", `${progress}%`);
    const disc = el("span", "maturity-disc", ring);
    disc.style.width = disc.style.height = `${size}px`;
    if (inside) {
      figure.style.fontSize = `${Math.round(size * 0.26)}px`;
      disc.append(figure);
    }
    const glyph = el("span", "maturity-glyph", disc);
    if (!inside) glyph.append(figure);
    glyph.setAttribute("aria-hidden", "true");
    setColors(glyph, bandOf(progress));
    return glyph;
  }

  function maturityRing(name, levels) {
    const progress = maturity(levels);
    const node = el("span", "maturity-ring", ringGlyph(progress, 18));
    node.setAttribute("role", "img");
    node.setAttribute("aria-label", `${name} maturity: ${progress}%, ${bandOf(progress).band}`);
    node.dataset.progress = progress;
    node.dataset.count = levels.length;
    return node;
  }

  // The popover, shared by every mark and ring, filled for whichever one
  // opened it.
  const details = el("div", "stability-details");
  details.id = "stability-details";
  details.setAttribute("role", "tooltip");
  details.hidden = true;
  document.body.append(details);

  function link(href, text) {
    const node = el("a", "stability-details-link", `${text} ↗`);
    node.href = href;
    node.target = "_blank";
    node.rel = "noopener noreferrer";
    return node;
  }

  // fillStability says how stable a mark's level is, like StabilityDetails.
  function fillStability(target) {
    const step = stepOf(target.dataset.stability);
    const level = LEVELS[step - 1];
    const { feature } = target.dataset;
    setColors(details, level);
    const levels = el(
      "div",
      "stability-details-levels",
      el("div", "stability-details-boxes", ...cells("", step)),
      el(
        "div",
        "stability-details-names",
        ...LEVELS.map((l) => el("span", l === level ? "current" : "", l.name)),
      ),
    );
    levels.setAttribute("aria-hidden", "true");
    details.replaceChildren(
      el("div", "stability-details-eyebrow", "Stability"),
      el("div", "stability-details-title", level.fullName, el("span", "", level.stability)),
      levels,
      ...(feature
        ? [el("div", "stability-details-statement", el("strong", "", feature), ` is in ${level.fullName}.`)]
        : []),
      el("div", feature ? "stability-details-definition" : "stability-details-summary", level.summary),
      el("div", "stability-details-about", STABILITY_ABOUT),
      link(`${STABILITY_DOCS_URL}#stability-${level.id}`, "How stability levels work"),
    );
  }

  // fillMaturity says how mature a ring's tag is, like MaturityDetails.
  function fillMaturity(target) {
    const progress = Number(target.dataset.progress);
    const count = Number(target.dataset.count);
    const band = bandOf(progress);
    setColors(details, band);
    details.replaceChildren(
      el("div", "stability-details-eyebrow", "Maturity"),
      el(
        "div",
        "maturity-details-head",
        ringGlyph(progress, 56),
        el(
          "div",
          "",
          el("div", "maturity-details-band", band.band),
          el("div", "maturity-details-basis", `Based on its ${count} ${count === 1 ? "endpoint" : "endpoints"}`),
        ),
      ),
      el("div", "stability-details-summary", band.bandSummary),
      el("div", "stability-details-about", MATURITY_ABOUT),
      link(MATURITY_DOCS_URL, "How maturity works"),
    );
  }

  let openFor = null;
  let closeTimer = 0;

  // place puts the popover below its target, or above it where there's no
  // room below, centred on it and kept 12px inside the window.
  function place(target) {
    const gap = 8;
    const margin = 12;
    // The window less its scrollbars.
    const { clientWidth, clientHeight } = document.documentElement;
    const anchor = target.getBoundingClientRect();
    const { width, height } = details.getBoundingClientRect();
    let top = anchor.bottom + gap;
    if (top + height > clientHeight - margin && anchor.top - gap - height >= margin) {
      top = anchor.top - gap - height;
    }
    const centred = anchor.left + anchor.width / 2 - width / 2;
    const left = Math.min(Math.max(centred, margin), clientWidth - width - margin);
    details.style.top = `${top}px`;
    details.style.left = `${left}px`;
  }

  function open(target) {
    clearTimeout(closeTimer);
    if (openFor === target) return;
    if (openFor) openFor.removeAttribute("aria-describedby");
    if (target.classList.contains("maturity-ring")) fillMaturity(target);
    else fillStability(target);
    // The popover sits outside Scalar's app, so it takes Scalar's font from
    // its target.
    details.style.fontFamily = getComputedStyle(target).fontFamily;
    details.hidden = false;
    place(target);
    target.setAttribute("aria-describedby", details.id);
    openFor = target;
  }

  function close() {
    clearTimeout(closeTimer);
    if (!openFor) return;
    openFor.removeAttribute("aria-describedby");
    details.hidden = true;
    openFor = null;
  }

  // closeSoon leaves time to move the pointer from the target into the
  // popover, so its link can be followed.
  function closeSoon() {
    clearTimeout(closeTimer);
    closeTimer = setTimeout(close, 150);
  }

  const targetOf = (event) => event.target.closest?.(".stability-mark, .maturity-ring");

  document.addEventListener("pointerover", (event) => {
    const target = targetOf(event);
    if (target && event.pointerType === "mouse") open(target);
    else if (details.contains(event.target)) clearTimeout(closeTimer);
  });
  document.addEventListener("pointerout", (event) => {
    if (targetOf(event) || details.contains(event.target)) closeSoon();
  });
  document.addEventListener("focusin", (event) => {
    const target = targetOf(event);
    if (target) open(target);
    else if (!details.contains(event.target)) close();
  });
  // A tap opens the popover, and a tap anywhere else closes it.
  document.addEventListener("click", (event) => {
    const target = targetOf(event);
    if (target) open(target);
    else if (!details.contains(event.target)) close();
  });
  document.addEventListener("keydown", (event) => {
    if (event.key === "Escape") close();
  });
  window.addEventListener("scroll", close, true);
  window.addEventListener("resize", close);

  // upgradeBadges replaces the text of every marked badge under root with a
  // long stability mark, and strips the pill Scalar drew around it.
  function upgradeBadges(root) {
    const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT);
    const marked = [];
    for (let node = walker.nextNode(); node; node = walker.nextNode()) {
      if (node.nodeValue.includes(STABILITY_MARK)) marked.push(node);
    }
    for (const node of marked) {
      const name = node.nodeValue.replace(STABILITY_MARK, "").trim();
      const level = LEVELS.find((l) => l.name === name);
      const host = node.parentElement;
      if (!level || !host) continue;
      host.style.background = "none";
      host.style.border = "0";
      host.style.boxShadow = "none";
      host.style.padding = "0";
      const replacement = mark(level);
      replacement.tabIndex = 0;
      node.replaceWith(replacement);
    }
  }

  // upgradeNav marks each operation in the navigation after its name, and
  // puts each tag's maturity beside its name, both outside the name, so a
  // long name wraps without leaving the mark on a line of its own. Scalar links an operation to
  // "#tag/<tag>/<METHOD>/<path>" and a tag to "#tag/<tag>". Neither takes
  // focus: each is inside a link, which already does.
  function upgradeNav(root) {
    for (const item of root.querySelectorAll('li[data-sidebar-id] > a[href^="#tag/"], li[data-sidebar-id] > div > a[href^="#tag/"]')) {
      if (item.querySelector(".stability-mark, .maturity-ring")) continue;
      const label = item.querySelector('[class~="group/button-label"]');
      if (!label) continue;
      const name = label.textContent.trim();
      const [, , method, ...path] = item.getAttribute("href").split("/");
      if (method) {
        const level = LEVELS.find((l) => l.id === operations[`${method} /${path.join("/")}`]);
        if (level && level.id !== "ga") {
          label.after(mark(level, { short: true, feature: name }));
        }
      } else if (tags[name]?.length) {
        label.after(maturityRing(name, tags[name]));
      }
    }
  }

  function upgrade() {
    upgradeBadges(document.body);
    upgradeNav(document.body);
  }

  // Scalar renders, and re-renders as you navigate, after this runs, so
  // badges and navigation items are upgraded as they appear.
  new MutationObserver(upgrade).observe(document.body, {
    childList: true,
    subtree: true,
    characterData: true,
  });
  upgrade();
})();
