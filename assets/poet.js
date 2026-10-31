// The page works without this file, as links and forms. It adds what needs a
// browser: the margin moves live, and the download and copy buttons.
(() => {
  const stage = document.getElementById("stage");
  if (!stage) return;

  const pad = document.getElementById("pad");
  pad.addEventListener("input", () => {
    stage.className = stage.className.replace(/pad-\d+/, "pad-" + pad.value);
    const update = (href) => {
      const u = new URL(href, location.href);
      u.searchParams.set("pad", pad.value);
      return u.pathname + u.search;
    };
    history.replaceState(null, "", update(location.href));
    document.querySelectorAll(".pills a, .swatches a").forEach((a) => {
      a.setAttribute("href", update(a.getAttribute("href")));
    });
  });

  const problem = document.getElementById("problem");
  const options = { pixelRatio: 2, style: { borderRadius: "0", margin: "0" } };

  const save = (href, name) => {
    const link = document.createElement("a");
    link.download = name;
    link.href = href;
    link.click();
  };

  const exporter = (button, run) => button.addEventListener("click", async () => {
    button.disabled = true;
    problem.hidden = true;
    try {
      await document.fonts.ready;
      await run();
    } catch (e) {
      console.error(e);
      problem.textContent = button.dataset.error;
      problem.hidden = false;
    }
    button.disabled = false;
  });

  // One page holding one JPEG, which is all a PDF needs for this. Text becomes
  // part of the picture.
  const pdfFromJpeg = (bytes, width, height) => {
    const text = new TextEncoder();
    const parts = [];
    const offsets = [];
    let length = 0;
    const add = (part) => {
      const chunk = typeof part === "string" ? text.encode(part) : part;
      parts.push(chunk);
      length += chunk.length;
    };
    const object = (n, body) => {
      offsets[n] = length;
      add(n + " 0 obj\n" + body + "\nendobj\n");
    };
    const w = (width / options.pixelRatio).toFixed(2);
    const h = (height / options.pixelRatio).toFixed(2);
    const draw = "q " + w + " 0 0 " + h + " 0 0 cm /Im0 Do Q";
    add("%PDF-1.4\n");
    object(1, "<< /Type /Catalog /Pages 2 0 R >>");
    object(2, "<< /Type /Pages /Kids [3 0 R] /Count 1 >>");
    object(3, "<< /Type /Page /Parent 2 0 R /MediaBox [0 0 " + w + " " + h + "] /Contents 4 0 R /Resources << /XObject << /Im0 5 0 R >> >> >>");
    object(4, "<< /Length " + draw.length + " >>\nstream\n" + draw + "\nendstream");
    offsets[5] = length;
    add("5 0 obj\n<< /Type /XObject /Subtype /Image /Width " + width + " /Height " + height +
      " /ColorSpace /DeviceRGB /BitsPerComponent 8 /Filter /DCTDecode /Length " + bytes.length + " >>\nstream\n");
    add(bytes);
    add("\nendstream\nendobj\n");
    const xref = length;
    let table = "xref\n0 6\n0000000000 65535 f \n";
    for (let n = 1; n <= 5; n++) table += String(offsets[n]).padStart(10, "0") + " 00000 n \n";
    add(table + "trailer\n<< /Size 6 /Root 1 0 R >>\nstartxref\n" + xref + "\n%%EOF\n");
    return new Blob(parts, { type: "application/pdf" });
  };

  const buildPdf = async () => {
    const canvas = await htmlToImage.toCanvas(stage, options);
    const jpeg = canvas.toDataURL("image/jpeg", 0.92).split(",")[1];
    const bytes = Uint8Array.from(atob(jpeg), (c) => c.charCodeAt(0));
    return pdfFromJpeg(bytes, canvas.width, canvas.height);
  };
  window.poet = { buildPdf, options }; // lets the browser test reach the output

  const png = document.getElementById("download");
  exporter(png, async () => save(await htmlToImage.toPng(stage, options), png.dataset.file));

  const pdf = document.getElementById("download-pdf");
  exporter(pdf, async () => {
    const url = URL.createObjectURL(await buildPdf());
    save(url, pdf.dataset.file);
    setTimeout(() => URL.revokeObjectURL(url), 60000);
  });

  const copy = document.getElementById("copy");
  const label = copy.textContent;
  copy.addEventListener("click", async () => {
    await navigator.clipboard.writeText(document.getElementById("alt").value);
    copy.textContent = copy.dataset.done;
    setTimeout(() => { copy.textContent = label; }, 1500);
  });
})();
