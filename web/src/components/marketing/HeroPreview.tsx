// Illustrative product preview shown in the hero. Static and decorative; the
// chart is aria-hidden and the card carries a text label for assistive tech.
export function HeroPreview() {
  // A small static sparkline (area + line) in the brand hue.
  const pts = [8, 14, 11, 20, 18, 28, 24, 34, 30, 42, 48, 44];
  const W = 320, H = 72;
  const max = Math.max(...pts);
  const step = W / (pts.length - 1);
  const line = pts.map((p, i) => `${i === 0 ? 'M' : 'L'}${(i * step).toFixed(1)},${(H - (p / max) * (H - 8) - 2).toFixed(1)}`).join(' ');
  const area = `${line} L${W},${H} L0,${H} Z`;

  return (
    <div className="preview" role="img" aria-label="Linkspire dashboard preview: a short link with privacy-safe analytics.">
      <div className="bar" aria-hidden="true"><i /><i /><i /></div>

      <div className="link-row">
        <div>
          <div className="code">lnk.spr/NoRuDiB</div>
          <div className="dest">→ example.com/launch</div>
        </div>
        <span className="badge ok"><span className="dot" aria-hidden="true" />clean</span>
      </div>

      <div className="mini-stats" aria-hidden="true">
        <div className="s"><div className="v">12,480</div><div className="k">Clicks</div></div>
        <div className="s"><div className="v">8,210</div><div className="k">Unique / day</div></div>
        <div className="s"><div className="v">42</div><div className="k">Countries</div></div>
      </div>

      <div className="spark" aria-hidden="true">
        <svg viewBox={`0 0 ${W} ${H}`} width="100%" height="72" preserveAspectRatio="none">
          <path d={area} fill="var(--accent-soft)" opacity="0.7" />
          <path d={line} fill="none" stroke="var(--c1)" strokeWidth="2" />
        </svg>
      </div>

      <div className="bar-list" aria-hidden="true" style={{ marginTop: 12 }}>
        <div className="row"><span className="k">United States</span><span className="v">41%</span><span className="track"><span className="fill" style={{ width: '41%' }} /></span></div>
        <div className="row"><span className="k">Germany</span><span className="v">23%</span><span className="track"><span className="fill" style={{ width: '23%' }} /></span></div>
        <div className="row"><span className="k">Japan</span><span className="v">16%</span><span className="track"><span className="fill" style={{ width: '16%' }} /></span></div>
      </div>
    </div>
  );
}
