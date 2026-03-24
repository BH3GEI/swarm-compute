// Job API — Distributed Computing
const CENTER = window.CENTER_BASE_URL || `${location.protocol}//${location.hostname}:8080`;

async function apiGetTaskTypes() {
  const r = await fetch(`${CENTER}/api/task-types`);
  return r.json();
}

async function apiSubmitJob(typeId, input, opts = {}) {
  const body = {
    typeId,
    input,
    maxRetry: opts.maxRetry || 2,
    timeoutSec: opts.timeoutSec || 120,
  };
  const r = await fetch(`${CENTER}/api/jobs/submit`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  });
  return r.json();
}

async function apiGetJob(jobId) {
  const r = await fetch(`${CENTER}/api/jobs/${jobId}`);
  return r.json();
}

async function apiListJobs() {
  const r = await fetch(`${CENTER}/api/jobs`);
  return r.json();
}

async function apiCancelJob(jobId) {
  const r = await fetch(`${CENTER}/api/jobs/${jobId}/cancel`, { method: 'POST' });
  return r.json();
}

async function apiGetStats() {
  const r = await fetch(`${CENTER}/api/stats`);
  return r.json();
}
