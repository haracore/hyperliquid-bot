package app

const pageTemplate = `
{{define "layout"}}
<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{.Title}} · Hyperliquid UI</title>
  <link rel="stylesheet" href="/static/app.css">
</head>
<body>
  <header class="topbar">
    <div>
      <div class="brand">Hyperliquid UI</div>
      <div class="meta">{{.BaseURL}}{{if .Testnet}} · testnet{{end}}{{if .PrivateKeySet}} · trading enabled{{else}} · read-only{{end}}</div>
    </div>
    <nav>
      <a class="{{if eq .Active "balances"}}active{{end}}" href="/balances">Balances</a>
      <a class="{{if eq .Active "positions"}}active{{end}}" href="/positions">Positions</a>
      <a class="{{if eq .Active "perp-orders"}}active{{end}}" href="/perp/orders">Perp Orders</a>
      <a class="{{if eq .Active "spot-orders"}}active{{end}}" href="/spot/orders">Spot Orders</a>
    </nav>
  </header>

  <main>
    {{if .Error}}<div class="notice error">{{.Error}}</div>{{end}}
    {{if eq .Active "balances"}}{{template "balances" .}}{{end}}
    {{if eq .Active "positions"}}{{template "positions" .}}{{end}}
    {{if eq .Active "perp-orders"}}{{template "orders" .}}{{end}}
    {{if eq .Active "spot-orders"}}{{template "orders" .}}{{end}}
    {{if .ResultJSON}}
      <section class="band">
        <div class="section-title">Raw Response</div>
        <pre>{{.ResultJSON}}</pre>
      </section>
    {{end}}
  </main>
</body>
</html>
{{end}}

{{define "balances"}}
<section class="band">
  <div class="section-title">Balances</div>
  <form method="post" class="toolbar">
    <label>Address <input name="address" value="{{.DefaultAddress}}" placeholder="0x..."></label>
    <button type="submit">Load</button>
  </form>
</section>

{{if .Balances}}
<section class="grid two">
  <div>
    <div class="section-title">Perp Account</div>
    {{if .Balances.PerpSummary}}
      <table><tbody>
      {{range .Balances.PerpSummary}}
        <tr><th>{{.Key}}</th><td>{{.Value}}</td></tr>
      {{end}}
      </tbody></table>
    {{else}}
      <div class="empty">No account summary returned.</div>
    {{end}}
  </div>

  <div>
    <div class="section-title">Spot Balances</div>
    {{if .Balances.SpotBalances}}
      <table>
        <thead><tr><th>Coin</th><th>Total</th><th>Hold</th></tr></thead>
        <tbody>
        {{range .Balances.SpotBalances}}
          <tr><td>{{.Coin}}</td><td>{{.Total}}</td><td>{{.Hold}}</td></tr>
        {{end}}
        </tbody>
      </table>
    {{else}}
      <div class="empty">No spot balances returned.</div>
    {{end}}
  </div>
</section>

<section class="band">
  <div class="section-title">Perp Positions</div>
  {{template "positionsTable" .Balances.PerpPositions}}
</section>
{{end}}
{{end}}

{{define "positions"}}
<section class="band">
  <div class="section-title">Perp Positions</div>
  <form method="post" class="toolbar">
    <label>Address <input name="address" value="{{.DefaultAddress}}" placeholder="0x..."></label>
    <label>Dex <input name="dex" value="{{.OrderContext.Dex}}" placeholder="default"></label>
    <label class="check"><input type="checkbox" name="all"> Show zero positions</label>
    <button type="submit">Load</button>
  </form>
</section>

{{if .Positions}}
<section class="band">
  {{template "positionsTable" .Positions}}
</section>
{{end}}
{{end}}

{{define "positionsTable"}}
{{if .}}
  <table>
    <thead><tr><th>Coin</th><th>Szi</th><th>Entry Px</th><th>Value</th><th>Unrealized PnL</th><th>Margin</th><th>Leverage</th></tr></thead>
    <tbody>
    {{range .}}
      <tr>
        <td>{{.Coin}}</td>
        <td>{{.Szi}}</td>
        <td>{{.EntryPx}}</td>
        <td>{{.PositionValue}}</td>
        <td>{{.UnrealizedPnl}}</td>
        <td>{{.MarginUsed}}</td>
        <td>{{.Leverage}}</td>
      </tr>
    {{end}}
    </tbody>
  </table>
{{else}}
  <div class="empty">No positions returned.</div>
{{end}}
{{end}}

{{define "orders"}}
<section class="band">
  <div class="section-title">{{.OrderContext.Kind}} Open Orders</div>
  <form method="post" class="toolbar">
    <input type="hidden" name="action" value="open-orders">
    <label>Address <input name="address" value="{{.DefaultAddress}}" placeholder="0x..."></label>
    {{if eq .OrderContext.Kind "perp"}}<label>Dex <input name="dex" value="{{.OrderContext.Dex}}" placeholder="default"></label>{{end}}
    {{if eq .OrderContext.Kind "spot"}}<label class="check"><input type="checkbox" name="frontend" checked> Frontend response</label>{{end}}
    <button type="submit">Load</button>
  </form>
</section>

<section class="grid two">
  <div>
    <div class="section-title">Create {{.OrderContext.Kind}} Order</div>
    <form method="post" class="stack">
      <input type="hidden" name="action" value="place">
      {{template "tradeFields" .}}
      <label>CLOID <input name="cloid" placeholder="0x00000000000000000000000000000001"></label>
      {{if eq .OrderContext.Kind "perp"}}<label class="check"><input type="checkbox" name="reduceOnly"> Reduce only</label>{{end}}
      {{template "confirm" .}}
      <button type="submit">Submit Order</button>
    </form>
  </div>

  <div>
    <div class="section-title">Modify {{.OrderContext.Kind}} Order</div>
    <form method="post" class="stack">
      <input type="hidden" name="action" value="modify">
      <label>OID <input name="oid" inputmode="numeric" placeholder="123"></label>
      <label>OID CLOID <input name="oidCloid" placeholder="optional existing cloid"></label>
      {{template "tradeFields" .}}
      <label>New CLOID <input name="newCloid" placeholder="optional new cloid"></label>
      {{if eq .OrderContext.Kind "perp"}}<label class="check"><input type="checkbox" name="reduceOnly"> Reduce only</label>{{end}}
      {{template "confirm" .}}
      <button type="submit">Modify Order</button>
    </form>
  </div>
</section>

<section class="grid two">
  <div>
    <div class="section-title">Cancel by OID</div>
    <form method="post" class="stack">
      <input type="hidden" name="action" value="cancel-oid">
      {{if eq .OrderContext.Kind "perp"}}<label>Dex <input name="dex" value="{{.OrderContext.Dex}}" placeholder="default"></label>{{end}}
      <label>Coin <input name="coin" placeholder="{{if eq .OrderContext.Kind "spot"}}PURR/USDC or @8{{else}}BTC{{end}}"></label>
      <label>OID <input name="oid" inputmode="numeric" placeholder="123"></label>
      {{template "confirm" .}}
      <button type="submit">Cancel</button>
    </form>
  </div>

  <div>
    <div class="section-title">Cancel by CLOID</div>
    <form method="post" class="stack">
      <input type="hidden" name="action" value="cancel-cloid">
      {{if eq .OrderContext.Kind "perp"}}<label>Dex <input name="dex" value="{{.OrderContext.Dex}}" placeholder="default"></label>{{end}}
      <label>Coin <input name="coin" placeholder="{{if eq .OrderContext.Kind "spot"}}PURR/USDC or @8{{else}}BTC{{end}}"></label>
      <label>CLOID <input name="cloid" placeholder="0x00000000000000000000000000000001"></label>
      {{template "confirm" .}}
      <button type="submit">Cancel</button>
    </form>
  </div>
</section>
{{end}}

{{define "tradeFields"}}
{{if eq .OrderContext.Kind "perp"}}<label>Dex <input name="dex" value="{{.OrderContext.Dex}}" placeholder="default"></label>{{end}}
<label>Coin <input name="coin" placeholder="{{if eq .OrderContext.Kind "spot"}}PURR/USDC or @8{{else}}BTC{{end}}"></label>
<label>Side
  <select name="side">
    <option value="buy">Buy</option>
    <option value="sell">Sell</option>
  </select>
</label>
<label>Size <input name="size" inputmode="decimal" placeholder="0.001"></label>
<label>Price <input name="price" inputmode="decimal" placeholder="25000"></label>
<label>TIF
  <select name="tif">
    <option value="Gtc">Gtc</option>
    <option value="Ioc">Ioc</option>
    <option value="Alo">Alo</option>
  </select>
</label>
{{end}}

{{define "confirm"}}
<label class="check danger"><input type="checkbox" name="confirm"> Confirm state-changing action</label>
{{end}}
`

const stylesheet = `
:root {
  color-scheme: light;
  --bg: #f7f7f4;
  --text: #1f2933;
  --muted: #64707d;
  --line: #d9ded8;
  --panel: #ffffff;
  --accent: #0f766e;
  --accent-dark: #115e59;
  --danger: #b42318;
  --danger-bg: #fff1f0;
}

* {
  box-sizing: border-box;
}

body {
  margin: 0;
  color: var(--text);
  background: var(--bg);
  font: 14px/1.45 system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
}

.topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 24px;
  padding: 18px 28px;
  border-bottom: 1px solid var(--line);
  background: var(--panel);
  position: sticky;
  top: 0;
  z-index: 2;
}

.brand {
  font-size: 18px;
  font-weight: 700;
}

.meta {
  color: var(--muted);
  font-size: 12px;
  margin-top: 2px;
}

nav {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

nav a {
  color: var(--text);
  text-decoration: none;
  border: 1px solid var(--line);
  padding: 8px 10px;
  border-radius: 6px;
  background: #fbfbfa;
}

nav a.active {
  color: #ffffff;
  border-color: var(--accent);
  background: var(--accent);
}

main {
  max-width: 1180px;
  margin: 0 auto;
  padding: 24px;
}

.band,
.grid > div {
  background: var(--panel);
  border: 1px solid var(--line);
  border-radius: 8px;
  padding: 18px;
  margin-bottom: 18px;
}

.grid {
  display: grid;
  gap: 18px;
  margin-bottom: 18px;
}

.grid.two {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.section-title {
  font-size: 15px;
  font-weight: 700;
  margin-bottom: 14px;
}

.toolbar,
.stack {
  display: grid;
  gap: 12px;
}

.toolbar {
  grid-template-columns: minmax(220px, 1fr) minmax(160px, 220px) auto;
  align-items: end;
}

.stack {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

label {
  display: grid;
  gap: 5px;
  color: var(--muted);
  font-size: 12px;
  font-weight: 600;
}

input,
select {
  min-width: 0;
  width: 100%;
  color: var(--text);
  background: #ffffff;
  border: 1px solid var(--line);
  border-radius: 6px;
  padding: 9px 10px;
  font: inherit;
}

.check {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--text);
  min-height: 38px;
}

.check input {
  width: auto;
}

.danger {
  color: var(--danger);
}

button {
  align-self: end;
  min-height: 38px;
  border: 1px solid var(--accent);
  background: var(--accent);
  color: #ffffff;
  border-radius: 6px;
  padding: 9px 13px;
  font: inherit;
  font-weight: 700;
  cursor: pointer;
}

button:hover {
  background: var(--accent-dark);
}

table {
  width: 100%;
  border-collapse: collapse;
  overflow: hidden;
}

th,
td {
  text-align: left;
  border-bottom: 1px solid var(--line);
  padding: 9px 8px;
  vertical-align: top;
}

th {
  color: var(--muted);
  font-size: 12px;
  font-weight: 700;
}

pre {
  overflow: auto;
  margin: 0;
  padding: 14px;
  border-radius: 6px;
  background: #172026;
  color: #e6edf3;
  font-size: 12px;
}

.notice {
  border-radius: 8px;
  padding: 12px 14px;
  margin-bottom: 18px;
  border: 1px solid var(--line);
}

.notice.error {
  color: var(--danger);
  background: var(--danger-bg);
  border-color: #ffd5d2;
}

.empty {
  color: var(--muted);
  padding: 10px 0;
}

@media (max-width: 820px) {
  .topbar {
    align-items: flex-start;
    flex-direction: column;
  }

  .grid.two,
  .toolbar,
  .stack {
    grid-template-columns: 1fr;
  }

  main {
    padding: 16px;
  }
}
`
