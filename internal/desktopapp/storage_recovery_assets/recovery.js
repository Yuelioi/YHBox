const elements = {
  state: document.querySelector("#state"),
  layout: document.querySelector("#layout"),
  backup: document.querySelector("#backup"),
  space: document.querySelector("#space"),
  runs: document.querySelector("#runs"),
  message: document.querySelector("#message"),
  resume: document.querySelector("#resume"),
  rollback: document.querySelector("#rollback"),
  export: document.querySelector("#export"),
  blockerSection: document.querySelector("#blocker-section"),
  blockerName: document.querySelector("#blocker-name"),
  quarantine: document.querySelector("#quarantine"),
  quarantineSection: document.querySelector("#quarantine-section"),
  quarantineList: document.querySelector("#quarantine-list"),
}

let current = null
let busy = false

function formatBytes(value) {
  if (!Number.isFinite(value) || value <= 0) return "0 B"
  const units = ["B", "KiB", "MiB", "GiB", "TiB"]
  const index = Math.min(Math.floor(Math.log(value) / Math.log(1024)), units.length - 1)
  const scaled = value / 1024 ** index
  return `${scaled.toFixed(index === 0 ? 0 : 1)} ${units[index]}`
}

function render(view) {
  current = view
  const status = view.status || {}
  const plan = status.plan || {}
  const journal = status.journal || null
  const quarantine = status.quarantine || []
  const state = journal?.state || (plan.from === plan.to ? "current" : "preflight")

  elements.layout.textContent = `${plan.from || "?"} -> ${plan.to || "?"}`
  elements.backup.textContent = formatBytes(plan.estimatedBackupBytes)
  elements.space.textContent = `${formatBytes(plan.availableBytes)} 可用`
  elements.runs.textContent = `${plan.legacyRunRecords || 0} 条`
  elements.message.textContent = view.message || "迁移状态已读取。"
  elements.state.textContent = stateLabel(state)
  elements.state.dataset.tone = state === "committed" || state === "current" ? "ready" : "blocked"

  const blocker = journal?.blockedEntry || ""
  elements.blockerSection.hidden = !blocker
  elements.blockerName.textContent = blocker
  elements.quarantine.dataset.name = blocker

  elements.quarantineSection.hidden = quarantine.length === 0
  elements.quarantineList.replaceChildren(
    ...quarantine.map((record) => {
      const row = document.createElement("div")
      row.className = "quarantine-row"
      const name = document.createElement("code")
      name.textContent = record.name
      const size = document.createElement("span")
      size.textContent = formatBytes(record.bytes)
      const restore = document.createElement("button")
      restore.type = "button"
      restore.textContent = "恢复记录"
      restore.addEventListener("click", () => action("restore", record.name))
      row.append(name, size, restore)
      return row
    }),
  )

  elements.rollback.disabled = busy || !journal || journal.state === "committed"
  elements.resume.disabled = busy || state === "committed" || state === "current"
  elements.quarantine.disabled = busy || !blocker
  elements.export.disabled = busy
}

function stateLabel(state) {
  const labels = {
    prepared: "快照已准备",
    applying: "等待继续",
    verifying: "等待验证",
    "recovery-required": "需要恢复",
    committed: "迁移已提交",
    "rolled-back": "已回滚",
    current: "布局已是最新",
    preflight: "预检查受阻",
  }
  return labels[state] || state
}

async function load() {
  try {
    const response = await fetch("/api/status", { cache: "no-store" })
    if (!response.ok) throw new Error(`读取状态失败: HTTP ${response.status}`)
    render(await response.json())
  } catch (error) {
    elements.state.textContent = "读取失败"
    elements.state.dataset.tone = "blocked"
    elements.message.textContent = String(error)
  }
}

async function action(name, recordName = "") {
  if (busy) return
  busy = true
  if (current) render(current)
  elements.message.textContent = "正在执行。请不要关闭 Yotta。"
  try {
    const response = await fetch("/api/action", {
      method: "POST",
      cache: "no-store",
      headers: {
        "Content-Type": "application/json",
        "X-Yotta-Recovery": "1",
      },
      body: JSON.stringify({ action: name, name: recordName }),
    })
    if (!response.ok) throw new Error(`恢复操作失败: HTTP ${response.status}`)
    render(await response.json())
  } catch (error) {
    elements.message.textContent = String(error)
  } finally {
    busy = false
    if (current) render(current)
  }
}

elements.resume.addEventListener("click", () => action("resume"))
elements.rollback.addEventListener("click", () => action("rollback"))
elements.export.addEventListener("click", () => action("export"))
elements.quarantine.addEventListener("click", () => action("quarantine", elements.quarantine.dataset.name))

load()
