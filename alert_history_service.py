#!/usr/bin/env python3
"""
Alert History Service для Alertmanager webhook:
- POST /webhook — приём событий алертов (firing/resolved)
- GET /history — выдача истории алертов (фильтры: alertname, status, fingerprint, время)
- GET /report — аналитика по истории алертов
- Хранение истории в SQLite (stateful)
"""

import os
import json
from fastapi import FastAPI, Request, Query
from fastapi.responses import JSONResponse
from fastapi.middleware.cors import CORSMiddleware
import sqlite3
from datetime import datetime, timedelta
from typing import Optional, List
from fastapi import HTTPException
from fastapi.templating import Jinja2Templates
from fastapi import Form
from fastapi.responses import HTMLResponse
from prometheus_client import Counter, Gauge, Histogram, generate_latest, CONTENT_TYPE_LATEST
from fastapi.responses import Response
import threading
import time

DB_PATH = os.environ.get("ALERT_HISTORY_DB", "alert_history.sqlite3")
RETENTION_DAYS = int(os.environ.get("RETENTION_DAYS", "30"))

app = FastAPI(title="Alertmanager Alert History Service")

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# --- DB INIT ---
def init_db():
    conn = sqlite3.connect(DB_PATH)
    c = conn.cursor()
    c.execute('''
    CREATE TABLE IF NOT EXISTS alert_history (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        alertname TEXT,
        fingerprint TEXT,
        status TEXT,
        labels TEXT,
        startsAt TEXT,
        endsAt TEXT,
        updatedAt TEXT,
        raw_json TEXT,
        namespace TEXT
    )
    ''')
    # Миграция: добавляем колонку namespace, если её нет
    c.execute("PRAGMA table_info(alert_history)")
    columns = [row[1] for row in c.fetchall()]
    if 'namespace' not in columns:
        c.execute('ALTER TABLE alert_history ADD COLUMN namespace TEXT')
    c.execute('''
    CREATE INDEX IF NOT EXISTS idx_alertname ON alert_history(alertname)
    ''')
    c.execute('''
    CREATE INDEX IF NOT EXISTS idx_fingerprint ON alert_history(fingerprint)
    ''')
    c.execute('''
    CREATE INDEX IF NOT EXISTS idx_status ON alert_history(status)
    ''')
    c.execute('''
    CREATE INDEX IF NOT EXISTS idx_namespace ON alert_history(namespace)
    ''')
    conn.commit()
    conn.close()

init_db()

# --- HELPERS ---
def get_db():
    return sqlite3.connect(DB_PATH)

def event_is_duplicate(conn, alertname, fingerprint, status, labels, startsAt, endsAt):
    c = conn.cursor()
    c.execute('''
        SELECT id FROM alert_history WHERE alertname=? AND fingerprint=? AND status=? AND labels=? AND startsAt=? AND endsAt=? ORDER BY id DESC LIMIT 1
    ''', (alertname, fingerprint, status, labels, startsAt, endsAt))
    return c.fetchone() is not None

# --- PROMETHEUS METRICS ---
WEBHOOK_EVENTS = Counter('alert_history_webhook_events_total', 'Всего принятых событий webhook', ['status', 'alertname', 'namespace'])
WEBHOOK_ERRORS = Counter('alert_history_webhook_errors_total', 'Ошибки обработки webhook')
HISTORY_QUERIES = Counter('alert_history_history_queries_total', 'Запросы к истории')
REPORT_QUERIES = Counter('alert_history_report_queries_total', 'Запросы к аналитике')
DB_ALERTS = Gauge('alert_history_db_alerts', 'Количество алертов в базе')
REQUEST_LATENCY = Histogram('alert_history_request_latency_seconds', 'Время обработки запроса', ['endpoint'])

# --- PATCH DB INIT ---
def count_alerts():
    try:
        with get_db() as conn:
            c = conn.cursor()
            c.execute('SELECT COUNT(*) FROM alert_history')
            cnt = c.fetchone()[0]
            DB_ALERTS.set(cnt)
    except Exception:
        pass

def prometheus_updater():
    while True:
        count_alerts()
        import time
        time.sleep(30)

threading.Thread(target=prometheus_updater, daemon=True).start()

# --- WEBHOOK ENDPOINT ---
@app.post("/webhook")
async def webhook(request: Request):
    start = time.time()
    try:
        payload = await request.json()
        now = datetime.utcnow().isoformat()
        alerts = payload.get("alerts", [])
        saved = 0
        with get_db() as conn:
            for alert in alerts:
                alertname = alert.get("labels", {}).get("alertname", "unknown")
                namespace = alert.get("labels", {}).get("namespace", "")
                fingerprint = alert.get("fingerprint") or alert.get("generatorURL") or ""
                status = alert.get("status", "unknown")
                labels = json.dumps(alert.get("labels", {}), sort_keys=True)
                startsAt = alert.get("startsAt")
                endsAt = alert.get("endsAt")
                raw_json = json.dumps(alert, ensure_ascii=False)
                if event_is_duplicate(conn, alertname, fingerprint, status, labels, startsAt, endsAt):
                    continue
                conn.execute('''
                    INSERT INTO alert_history (alertname, fingerprint, status, labels, startsAt, endsAt, updatedAt, raw_json, namespace)
                    VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
                ''', (alertname, fingerprint, status, labels, startsAt, endsAt, now, raw_json, namespace))
                saved += 1
                WEBHOOK_EVENTS.labels(status=status, alertname=alertname, namespace=namespace).inc()
            conn.commit()
        REQUEST_LATENCY.labels(endpoint="/webhook").observe(time.time() - start)
        return {"result": "ok", "saved": saved}
    except Exception as e:
        WEBHOOK_ERRORS.inc()
        REQUEST_LATENCY.labels(endpoint="/webhook").observe(time.time() - start)
        raise

# --- HISTORY ENDPOINT ---
@app.get("/history")
async def history(
    alertname: Optional[str] = Query(None),
    status: Optional[str] = Query(None),
    fingerprint: Optional[str] = Query(None),
    namespace: Optional[str] = Query(None),
    since: Optional[str] = Query(None),  # ISO8601
    until: Optional[str] = Query(None),
    limit: int = Query(100, ge=1, le=1000),
    offset: int = Query(0, ge=0)
):
    start = time.time()
    HISTORY_QUERIES.inc()
    q = "SELECT alertname, fingerprint, status, labels, startsAt, endsAt, updatedAt, namespace FROM alert_history WHERE 1=1"
    params = []
    if alertname:
        q += " AND alertname=?"
        params.append(alertname)
    if status:
        q += " AND status=?"
        params.append(status)
    if fingerprint:
        q += " AND fingerprint=?"
        params.append(fingerprint)
    if namespace:
        q += " AND namespace=?"
        params.append(namespace)
    if since:
        q += " AND updatedAt>=?"
        params.append(since)
    if until:
        q += " AND updatedAt<=?"
        params.append(until)
    q += " ORDER BY updatedAt DESC LIMIT ? OFFSET ?"
    params.extend([limit, offset])
    with get_db() as conn:
        c = conn.cursor()
        c.execute(q, params)
        rows = c.fetchall()
    result = [
        {
            "alertname": r[0],
            "fingerprint": r[1],
            "status": r[2],
            "labels": json.loads(r[3]),
            "startsAt": r[4],
            "endsAt": r[5],
            "updatedAt": r[6],
            "namespace": r[7],
        }
        for r in rows
    ]
    REQUEST_LATENCY.labels(endpoint="/history").observe(time.time() - start)
    return JSONResponse(result)

# --- REPORT ENDPOINT ---
@app.get("/report")
async def report(
    alertname: Optional[str] = Query(None),
    namespace: Optional[str] = Query(None),
    since: Optional[str] = Query(None),
    until: Optional[str] = Query(None),
    min_flap: int = Query(2, ge=1, le=100),
    top: int = Query(10, ge=1, le=100)
):
    """
    Аналитика по истории алертов:
    - Топ-алерты по количеству событий
    - Flapping (кол-во смен статуса)
    - Суммарная статистика
    """
    start = time.time()
    REPORT_QUERIES.inc()
    q = "SELECT alertname, status, updatedAt, namespace FROM alert_history WHERE 1=1"
    params = []
    if alertname:
        q += " AND alertname=?"
        params.append(alertname)
    if namespace:
        q += " AND namespace=?"
        params.append(namespace)
    if since:
        q += " AND updatedAt>=?"
        params.append(since)
    if until:
        q += " AND updatedAt<=?"
        params.append(until)
    q += " ORDER BY alertname, updatedAt"
    with get_db() as conn:
        c = conn.cursor()
        c.execute(q, params)
        rows = c.fetchall()
    from collections import defaultdict, Counter
    flap_counter = Counter()
    event_counter = Counter()
    last_status = {}
    for alertname_, status, updatedAt, namespace_ in rows:
        key = (alertname_, namespace_)
        event_counter[key] += 1
        if key not in last_status:
            last_status[key] = status
            continue
        if last_status[key] != status:
            flap_counter[key] += 1
        last_status[key] = status
    flapping = [
        {"alertname": k[0], "namespace": k[1], "flaps": v}
        for k, v in flap_counter.items() if v >= min_flap
    ]
    flapping.sort(key=lambda x: x["flaps"], reverse=True)
    top_alerts = [ (k[0], k[1], v) for k, v in event_counter.most_common(top) ]
    summary = {
        "total_events": len(rows),
        "unique_alerts": len(event_counter),
        "top_alerts": [ {"alertname": k[0], "namespace": k[1], "events": v} for k, v in event_counter.most_common(top) ],
        "flapping_alerts": flapping[:top],
    }
    REQUEST_LATENCY.labels(endpoint="/report").observe(time.time() - start)
    return summary

# --- DASHBOARD ENDPOINT ---
templates = Jinja2Templates(directory="templates")

@app.get("/dashboard", response_class=HTMLResponse)
async def dashboard(request: Request,
                   alertname: Optional[str] = None,
                   namespace: Optional[str] = None,
                   since: Optional[str] = None,
                   until: Optional[str] = None,
                   min_flap: int = 2,
                   top: int = 10):
    q = "SELECT alertname, status, updatedAt, namespace FROM alert_history WHERE 1=1"
    params = []
    if alertname:
        q += " AND alertname=?"
        params.append(alertname)
    if namespace:
        q += " AND namespace=?"
        params.append(namespace)
    if since:
        q += " AND updatedAt>=?"
        params.append(since)
    if until:
        q += " AND updatedAt<=?"
        params.append(until)
    q += " ORDER BY alertname, updatedAt"
    with get_db() as conn:
        c = conn.cursor()
        c.execute(q, params)
        rows = c.fetchall()
    from collections import Counter
    flap_counter = Counter()
    event_counter = Counter()
    last_status = {}
    for alertname_, status, updatedAt, namespace_ in rows:
        key = (alertname_, namespace_)
        event_counter[key] += 1
        if key not in last_status:
            last_status[key] = status
            continue
        if last_status[key] != status:
            flap_counter[key] += 1
        last_status[key] = status
    flapping = [
        {"alertname": k[0], "namespace": k[1], "flaps": v}
        for k, v in flap_counter.items() if v >= min_flap
    ]
    flapping.sort(key=lambda x: x["flaps"], reverse=True)
    top_alerts = [ (k[0], k[1], v) for k, v in event_counter.most_common(top) ]
    return templates.TemplateResponse("dashboard.html", {
        "request": request,
        "top_alerts": top_alerts,
        "flapping": flapping[:top],
        "alertname": alertname or "",
        "namespace": namespace or "",
        "since": since or "",
        "until": until or "",
        "min_flap": min_flap,
        "top": top
    })

@app.get("/dashboard_grouped", response_class=HTMLResponse)
async def dashboard_grouped(request: Request,
                           since: Optional[str] = None,
                           until: Optional[str] = None,
                           top: int = 10):
    # Группировка по alertname (всего событий)
    q1 = "SELECT alertname, COUNT(*) as cnt FROM alert_history WHERE 1=1"
    params1 = []
    if since:
        q1 += " AND updatedAt>=?"
        params1.append(since)
    if until:
        q1 += " AND updatedAt<=?"
        params1.append(until)
    q1 += " GROUP BY alertname ORDER BY cnt DESC LIMIT ?"
    params1.append(top)
    # Группировка по alertname (уникальные инциденты)
    q1u = "SELECT alertname, COUNT(DISTINCT fingerprint || '|' || startsAt) as uniq_cnt FROM alert_history WHERE 1=1"
    params1u = []
    if since:
        q1u += " AND updatedAt>=?"
        params1u.append(since)
    if until:
        q1u += " AND updatedAt<=?"
        params1u.append(until)
    q1u += " GROUP BY alertname ORDER BY uniq_cnt DESC LIMIT ?"
    params1u.append(top)
    # Группировка по namespace (всего событий)
    q2 = "SELECT namespace, COUNT(*) as cnt FROM alert_history WHERE 1=1"
    params2 = []
    if since:
        q2 += " AND updatedAt>=?"
        params2.append(since)
    if until:
        q2 += " AND updatedAt<=?"
        params2.append(until)
    q2 += " GROUP BY namespace ORDER BY cnt DESC LIMIT ?"
    params2.append(top)
    # Группировка по namespace (уникальные инциденты)
    q2u = "SELECT namespace, COUNT(DISTINCT fingerprint || '|' || startsAt) as uniq_cnt FROM alert_history WHERE 1=1"
    params2u = []
    if since:
        q2u += " AND updatedAt>=?"
        params2u.append(since)
    if until:
        q2u += " AND updatedAt<=?"
        params2u.append(until)
    q2u += " GROUP BY namespace ORDER BY uniq_cnt DESC LIMIT ?"
    params2u.append(top)
    with get_db() as conn:
        c = conn.cursor()
        c.execute(q1, params1)
        alertname_stats = c.fetchall()
        c.execute(q1u, params1u)
        alertname_uniq_stats = c.fetchall()
        c.execute(q2, params2)
        namespace_stats = c.fetchall()
        c.execute(q2u, params2u)
        namespace_uniq_stats = c.fetchall()
    return templates.TemplateResponse("dashboard_grouped.html", {
        "request": request,
        "alertname_stats": alertname_stats,
        "alertname_uniq_stats": alertname_uniq_stats,
        "namespace_stats": namespace_stats,
        "namespace_uniq_stats": namespace_uniq_stats,
        "since": since or "",
        "until": until or "",
        "top": top
    })

# --- HEALTH ---
@app.get("/healthz")
async def healthz():
    return {"status": "ok"}

@app.post("/fill_namespaces")
def fill_namespaces():
    """
    Заполняет поле namespace для старых записей, если оно пустое, но есть в labels.
    """
    updated = 0
    with get_db() as conn:
        c = conn.cursor()
        c.execute("SELECT id, labels FROM alert_history WHERE namespace IS NULL OR namespace = ''")
        rows = c.fetchall()
        for row in rows:
            id_, labels_json = row
            try:
                labels = json.loads(labels_json)
                ns = labels.get("namespace", "")
                if ns:
                    c.execute("UPDATE alert_history SET namespace=? WHERE id=?", (ns, id_))
                    updated += 1
            except Exception:
                continue
        conn.commit()
    return {"updated": updated}

@app.get("/metrics")
def metrics():
    return Response(generate_latest(), media_type=CONTENT_TYPE_LATEST)

# --- CLEANUP FUNCTION ---
def cleanup_old_data():
    """Удаляет данные старше RETENTION_DAYS дней"""
    while True:
        try:
            cutoff_date = (datetime.utcnow() - timedelta(days=RETENTION_DAYS)).isoformat()
            with get_db() as conn:
                c = conn.cursor()
                c.execute("DELETE FROM alert_history WHERE updatedAt < ?", (cutoff_date,))
                deleted = c.rowcount
                conn.commit()
                if deleted > 0:
                    print(f"Cleaned up {deleted} old records")
        except Exception as e:
            print(f"Error during cleanup: {e}")
        time.sleep(3600)  # Проверяем каждый час

# Запускаем очистку в отдельном потоке
threading.Thread(target=cleanup_old_data, daemon=True).start()
