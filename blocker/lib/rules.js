
const RULE_ID_COUNTER_KEY = "ruleIdCounter";
const RULE_MAP_KEY = "ruleMap";

function signatureOf(entry) {
  return JSON.stringify([entry.confidence, entry.reason, entry.matchedFields]);
}

async function nextRuleId() {
  const { [RULE_ID_COUNTER_KEY]: counter = 0 } = await chrome.storage.local.get(RULE_ID_COUNTER_KEY);
  const next = counter + 1;
  await chrome.storage.local.set({ [RULE_ID_COUNTER_KEY]: next });
  return next;
}

function redirectUrl(entry) {
  const params = new URLSearchParams({
    d: entry.domain,
    c: String(entry.confidence ?? 0),
    r: entry.reason ?? "",
    m: JSON.stringify(entry.matchedFields ?? []),
    id: entry.id ?? "",
  });
  return `/blocked.html?${params.toString()}`;
}

function buildRule(ruleId, entry) {
  return {
    id: ruleId,
    priority: 1,
    action: {
      type: "redirect",
      redirect: { extensionPath: redirectUrl(entry) },
    },
    condition: {
      requestDomains: [entry.domain],
      resourceTypes: ["main_frame"],
    },
  };
}

export async function syncRules(entries) {
  const { [RULE_MAP_KEY]: ruleMap = {} } = await chrome.storage.local.get(RULE_MAP_KEY);

  const wantedDomains = new Set(entries.map((e) => e.domain));
  const removeRuleIds = [];
  for (const domain of Object.keys(ruleMap)) {
    if (!wantedDomains.has(domain)) {
      removeRuleIds.push(ruleMap[domain].ruleId);
      delete ruleMap[domain];
    }
  }

  const addRules = [];
  for (const entry of entries) {
    const signature = signatureOf(entry);
    const existing = ruleMap[entry.domain];
    if (existing && existing.signature === signature) continue;

    let ruleId = existing?.ruleId;
    if (ruleId) {
      removeRuleIds.push(ruleId);
    } else {
      ruleId = await nextRuleId();
    }
    ruleMap[entry.domain] = { ruleId, signature };
    addRules.push(buildRule(ruleId, entry));
  }

  if (removeRuleIds.length || addRules.length) {
    await chrome.declarativeNetRequest.updateDynamicRules({
      removeRuleIds,
      addRules,
    });
  }
  await chrome.storage.local.set({ [RULE_MAP_KEY]: ruleMap });
}

export async function ruleCount() {
  const { [RULE_MAP_KEY]: ruleMap = {} } = await chrome.storage.local.get(RULE_MAP_KEY);
  return Object.keys(ruleMap).length;
}
