const RULE_MAP_KEY = "ruleMap";

function signatureOf(entry) {
  return JSON.stringify([entry.confidence, entry.reason, entry.matchedFields]);
}

function ruleIdFor(domain) {
  let hash = 5381;
  for (let i = 0; i < domain.length; i++) {
    hash = ((hash << 5) + hash + domain.charCodeAt(i)) | 0;
  }
  return ((hash >>> 0) % 0x3fffffff) + 1;
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
      removeRuleIds.push(ruleIdFor(domain));
      delete ruleMap[domain];
    }
  }

  const addRules = [];
  for (const entry of entries) {
    const signature = signatureOf(entry);
    if (ruleMap[entry.domain] === signature) continue;

    const ruleId = ruleIdFor(entry.domain);
    removeRuleIds.push(ruleId);
    ruleMap[entry.domain] = signature;
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
