"""
Nexus Python Sidecar — :17893
Go 백엔드가 처리 못하는 AI/ML/Python 전용 기능 담당
"""
import os, sys, json, sqlite3, threading, time, base64, re, subprocess
from pathlib import Path
from typing import Optional, List, Dict, Any
from contextlib import asynccontextmanager

from fastapi import FastAPI, HTTPException, UploadFile, File
from fastapi.responses import JSONResponse, StreamingResponse
import uvicorn
import requests

# ── 데이터 디렉토리 ──────────────────────────────────────────
APP_DATA = Path(os.environ.get("APPDATA", Path.home())) / "Nexus"
APP_DATA.mkdir(parents=True, exist_ok=True)
DB_PATH  = APP_DATA / "nexus_python.db"

# ── DB 초기화 ────────────────────────────────────────────────
def init_db():
    con = sqlite3.connect(DB_PATH)
    cur = con.cursor()
    cur.executescript("""
    CREATE TABLE IF NOT EXISTS brain_docs (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        source TEXT, content TEXT, embedding BLOB,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP
    );
    CREATE TABLE IF NOT EXISTS memory (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        key TEXT UNIQUE, value TEXT,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP
    );
    CREATE TABLE IF NOT EXISTS stock_watchlist (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        symbol TEXT UNIQUE, name TEXT,
        added_at DATETIME DEFAULT CURRENT_TIMESTAMP
    );
    CREATE TABLE IF NOT EXISTS workflows (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        name TEXT, description TEXT, yaml TEXT,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP
    );
    """)
    con.commit(); con.close()

init_db()

# ── API 키 (Go 백엔드가 /admin/keys 로 주입) ─────────────────
GROQ_KEY   = os.environ.get("NEXUS_GROQ_KEY", "")
CLAUDE_KEY = os.environ.get("NEXUS_CLAUDE_KEY", "")
TAVILY_KEY = os.environ.get("NEXUS_TAVILY_KEY", "")

def _load_keys_from_config():
    """llm_config.json 에서 평문 키 로드 (Mac 개발환경 전용 — Windows는 Go가 주입)"""
    global GROQ_KEY, CLAUDE_KEY, TAVILY_KEY
    config_paths = [
        Path(os.environ.get("APPDATA", "")) / "Nexus" / "llm_config.json",
        Path.home() / ".nexus" / "llm_config.json",
    ]
    for p in config_paths:
        if p.exists():
            try:
                cfg = json.loads(p.read_text(encoding="utf-8"))
                if not GROQ_KEY:   GROQ_KEY   = cfg.get("groq_key", "")
                if not CLAUDE_KEY: CLAUDE_KEY = cfg.get("claude_key", "")
                if not TAVILY_KEY: TAVILY_KEY = cfg.get("tavily_key", "")
                break
            except Exception:
                pass

_load_keys_from_config()

def groq_chat(messages: list, model="llama-3.1-8b-instant", max_tokens=1024) -> str:
    if not GROQ_KEY:
        return ""
    try:
        r = requests.post(
            "https://api.groq.com/openai/v1/chat/completions",
            headers={"Authorization": f"Bearer {GROQ_KEY}", "Content-Type": "application/json"},
            json={"model": model, "messages": messages, "max_tokens": max_tokens},
            timeout=20
        )
        return r.json()["choices"][0]["message"]["content"]
    except Exception:
        return ""

# ── FastAPI ──────────────────────────────────────────────────
@asynccontextmanager
async def lifespan(app: FastAPI):
    yield

app = FastAPI(title="Nexus Python Sidecar", version="1.0.0", lifespan=lifespan)

def ok(**kwargs): return {"success": True, **kwargs}
def fail(msg): return {"success": False, "message": msg}

# ── Go 백엔드가 시작 직후 키를 주입하는 내부 엔드포인트 ──────────
@app.post("/admin/keys")
def admin_set_keys(body: dict):
    global GROQ_KEY, CLAUDE_KEY, TAVILY_KEY
    if body.get("groq_key"):   GROQ_KEY   = body["groq_key"]
    if body.get("claude_key"): CLAUDE_KEY = body["claude_key"]
    if body.get("tavily_key"): TAVILY_KEY = body["tavily_key"]
    return ok(message="keys updated", groq=bool(GROQ_KEY), claude=bool(CLAUDE_KEY), tavily=bool(TAVILY_KEY))

# ════════════════════════════════════════════════════════════
# 2단계 — 검색
# ════════════════════════════════════════════════════════════













# ════════════════════════════════════════════════════════════
# 3단계 — 문서/데이터
# ════════════════════════════════════════════════════════════

@app.post("/vision/ocr")
async def ocr_image(file: Optional[UploadFile] = File(None), body: dict = None):
    try:
        import easyocr
        reader = easyocr.Reader(["ko", "en"], gpu=False)
        if file:
            data = await file.read()
        else:
            return fail("이미지 파일 필요")
        import numpy as np
        from PIL import Image
        import io
        img = Image.open(io.BytesIO(data))
        result = reader.readtext(np.array(img))
        text = "\n".join(r[1] for r in result)
        return ok(text=text, blocks=len(result))
    except Exception as e:
        return fail(str(e))


@app.post("/vision/ocr-base64")
def ocr_base64(body: dict):
    b64 = body.get("image_base64", "")
    if not b64:
        return fail("image_base64 필요")
    try:
        import easyocr, numpy as np
        from PIL import Image
        import io
        data = base64.b64decode(b64)
        img = Image.open(io.BytesIO(data))
        reader = easyocr.Reader(["ko", "en"], gpu=False)
        result = reader.readtext(np.array(img))
        text = "\n".join(r[1] for r in result)
        return ok(text=text, blocks=len(result), message="OCR 완료")
    except Exception as e:
        return fail(str(e))


@app.post("/docs/pdf-extract")
def pdf_extract(body: dict):
    path = body.get("path", "")
    extract_tables = body.get("extract_tables", False)
    if not path or not os.path.exists(path):
        return fail("파일 없음")
    try:
        import fitz
        doc = fitz.open(path)
        pages = []
        for i, page in enumerate(doc):
            text = page.get_text()
            images = []
            for img in page.get_images():
                xref = img[0]
                base_image = doc.extract_image(xref)
                images.append({"width": base_image["width"], "height": base_image["height"]})
            pages.append({"page": i+1, "text": text, "images": len(images)})
        tables = []
        if extract_tables:
            for page in doc:
                tabs = page.find_tables()
                for tab in tabs.tables:
                    tables.append(tab.extract())
        full_text = "\n\n".join(p["text"] for p in pages)
        return ok(pages=pages, tables=tables, full_text=full_text,
                  page_count=len(pages), message=f"PDF {len(pages)}페이지 추출 완료")
    except Exception as e:
        return fail(str(e))


@app.post("/excel/read")
def excel_read(body: dict):
    path = body.get("path", "")
    sheet = body.get("sheet", 0)
    if not path or not os.path.exists(path):
        return fail("파일 없음")
    try:
        import pandas as pd
        if isinstance(sheet, int):
            df = pd.read_excel(path, sheet_name=sheet)
        else:
            df = pd.read_excel(path, sheet_name=sheet)
        stats = {
            "rows": len(df), "cols": len(df.columns),
            "columns": df.columns.tolist(),
            "summary": df.describe(include="all").to_dict(),
            "null_counts": df.isnull().sum().to_dict(),
        }
        preview = df.head(20).fillna("").to_dict(orient="records")
        return ok(preview=preview, stats=stats, message=f"Excel {len(df)}행 로드 완료")
    except Exception as e:
        return fail(str(e))


@app.post("/excel/save")
def excel_save(body: dict):
    path = body.get("path", "")
    data = body.get("data", [])
    sheet_name = body.get("sheet_name", "Sheet1")
    if not path:
        return fail("path 필요")
    try:
        import pandas as pd
        df = pd.DataFrame(data)
        df.to_excel(path, index=False, sheet_name=sheet_name)
        return ok(path=path, rows=len(df), message=f"Excel 저장 완료: {path}")
    except Exception as e:
        return fail(str(e))


@app.post("/screenshot/analyze")
def screenshot_analyze(body: dict):
    image_base64 = body.get("image_base64", "")
    question = body.get("question", "이 화면에서 무엇을 볼 수 있나요?")
    claude_key = body.get("claude_key", "") or CLAUDE_KEY
    if not image_base64:
        return fail("image_base64 필요")
    if not claude_key:
        return fail("Claude API 키 필요")
    try:
        r = requests.post(
            "https://api.anthropic.com/v1/messages",
            headers={"x-api-key": claude_key, "anthropic-version": "2023-06-01",
                     "Content-Type": "application/json"},
            json={"model": "claude-haiku-4-5-20251001", "max_tokens": 1024,
                  "messages": [{"role": "user", "content": [
                      {"type": "image", "source": {"type": "base64", "media_type": "image/png",
                                                   "data": image_base64}},
                      {"type": "text", "text": question}
                  ]}]},
            timeout=30
        )
        text = r.json()["content"][0]["text"]
        return ok(analysis=text, message="화면 분석 완료")
    except Exception as e:
        return fail(str(e))




# ════════════════════════════════════════════════════════════
# 4단계 — Email AI
# ════════════════════════════════════════════════════════════

















# ════════════════════════════════════════════════════════════
# 5단계 — Brain/Memory
# ════════════════════════════════════════════════════════════

_encoder = None
_index = None
_index_ids = []

def get_encoder():
    global _encoder
    if _encoder is None:
        from sentence_transformers import SentenceTransformer
        _encoder = SentenceTransformer("paraphrase-multilingual-MiniLM-L12-v2")
    return _encoder

def get_index():
    global _index, _index_ids
    if _index is None:
        import faiss, numpy as np
        _index = faiss.IndexFlatL2(384)
        con = sqlite3.connect(DB_PATH)
        rows = con.execute("SELECT id, embedding FROM brain_docs WHERE embedding IS NOT NULL").fetchall()
        con.close()
        if rows:
            ids, vecs = [], []
            for row_id, emb_blob in rows:
                if emb_blob:
                    vec = np.frombuffer(emb_blob, dtype=np.float32)
                    if vec.shape[0] == 384:
                        ids.append(row_id); vecs.append(vec)
            if vecs:
                _index.add(np.array(vecs))
                _index_ids = ids
    return _index, _index_ids


@app.post("/brain/index")
def brain_index(body: dict):
    source  = body.get("source", "")
    content = body.get("content", "")
    if not content:
        return fail("content 필요")
    try:
        import numpy as np
        enc = get_encoder()
        vec = enc.encode([content])[0].astype(np.float32)
        con = sqlite3.connect(DB_PATH)
        con.execute("INSERT INTO brain_docs (source, content, embedding) VALUES (?, ?, ?)",
                    (source, content, vec.tobytes()))
        con.commit(); con.close()
        global _index, _index_ids
        _index = None  # 재빌드 트리거
        return ok(message="인덱싱 완료")
    except Exception as e:
        return fail(str(e))


@app.post("/brain/search")
def brain_search(body: dict):
    query = body.get("query", "")
    top_k = body.get("top_k", 5)
    if not query:
        return fail("query 필요")
    try:
        import numpy as np
        enc = get_encoder()
        idx, ids = get_index()
        if idx.ntotal == 0:
            # 인덱스 비어있음 → Groq + Tavily 웹 폴백
            fallback_answer = ""
            web_items = tavily_search_local(query, 3)
            if web_items:
                context = "\n".join(
                    f"- {r.get('title','')}: {r.get('content','')[:200]}"
                    for r in web_items
                )
                fallback_answer = groq_chat(
                    [{"role": "user", "content": f"다음 정보를 바탕으로 '{query}'에 대해 답해줘:\n{context}"}],
                    max_tokens=500
                )
            elif GROQ_KEY:
                fallback_answer = groq_chat(
                    [{"role": "user", "content": query}], max_tokens=400
                )
            msg = "🧠 Second Brain 인덱스가 비어있어요. 파일을 인덱싱하면 개인화 검색이 가능해요."
            if fallback_answer:
                msg = f"🧠 Second Brain이 비어있어 웹 검색으로 대체했어요:\n\n{fallback_answer}"
            return ok(results=[], count=0, fallback=fallback_answer, message=msg)
        qvec = enc.encode([query])[0].astype(np.float32).reshape(1, -1)
        distances, indices = idx.search(qvec, min(top_k, idx.ntotal))
        con = sqlite3.connect(DB_PATH)
        results = []
        for dist, i in zip(distances[0], indices[0]):
            if i < 0 or i >= len(ids):
                continue
            row = con.execute("SELECT source, content FROM brain_docs WHERE id=?", (ids[i],)).fetchone()
            if row:
                results.append({"source": row[0], "content": row[1][:300], "score": float(1 / (1 + dist))})
        con.close()
        return ok(results=results, count=len(results), message=f"'{query}' 관련 {len(results)}개 발견")
    except Exception as e:
        return fail(str(e))


@app.get("/brain/stats")
def brain_stats():
    try:
        con = sqlite3.connect(DB_PATH)
        total = con.execute("SELECT COUNT(*) FROM brain_docs").fetchone()[0]
        sources = con.execute("SELECT COUNT(DISTINCT source) FROM brain_docs").fetchone()[0]
        con.close()
        return ok(total_docs=total, unique_sources=sources, message=f"총 {total}개 문서 인덱싱됨")
    except Exception as e:
        return fail(str(e))


@app.post("/brain/rebuild")
def brain_rebuild():
    global _index, _index_ids
    _index = None; _index_ids = []
    try:
        get_index()
        return ok(message="인덱스 재빌드 완료")
    except Exception as e:
        return fail(str(e))


@app.get("/memory/list")
def memory_list():
    con = sqlite3.connect(DB_PATH)
    rows = con.execute("SELECT key, value, created_at FROM memory ORDER BY created_at DESC").fetchall()
    con.close()
    entries = [{"key": r[0], "value": r[1], "created_at": r[2]} for r in rows]
    return ok(entries=entries, total=len(entries))


@app.post("/memory/search")
def memory_search(body: dict):
    q = body.get("query", "").lower()
    con = sqlite3.connect(DB_PATH)
    rows = con.execute("SELECT key, value FROM memory WHERE key LIKE ? OR value LIKE ?",
                       (f"%{q}%", f"%{q}%")).fetchall()
    con.close()
    results = [{"key": r[0], "value": r[1]} for r in rows]
    return ok(results=results, count=len(results))


@app.post("/memory/save")
def memory_save(body: dict):
    key = body.get("key", ""); value = body.get("value", "")
    if not key:
        return fail("key 필요")
    con = sqlite3.connect(DB_PATH)
    con.execute("INSERT OR REPLACE INTO memory (key, value) VALUES (?, ?)", (key, value))
    con.commit(); con.close()
    return ok(message="저장 완료")


@app.get("/memory/stats")
def memory_stats():
    con = sqlite3.connect(DB_PATH)
    total = con.execute("SELECT COUNT(*) FROM memory").fetchone()[0]
    con.close()
    return ok(total=total)


@app.post("/memory/clear")
def memory_clear():
    con = sqlite3.connect(DB_PATH)
    con.execute("DELETE FROM memory"); con.commit(); con.close()
    return ok(message="메모리 초기화 완료")


# ════════════════════════════════════════════════════════════
# 6단계 — 주식/보안/웹
# ════════════════════════════════════════════════════════════



















# ════════════════════════════════════════════════════════════
# 7단계 — Desktop Agent
# ════════════════════════════════════════════════════════════

_desktop_task: Optional[threading.Thread] = None
_desktop_cancel_flag = threading.Event()


@app.post("/desktop/click")
def desktop_click(body: dict):
    x = body.get("x"); y = body.get("y")
    button = body.get("button", "left")
    clicks = body.get("clicks", 1)
    if x is None or y is None:
        return fail("x, y 필요")
    try:
        import pyautogui
        pyautogui.click(x, y, clicks=clicks, button=button)
        return ok(message=f"클릭: ({x}, {y})")
    except Exception as e:
        return fail(str(e))


@app.post("/desktop/type")
def desktop_type(body: dict):
    text    = body.get("text", "")
    interval = body.get("interval", 0.02)
    if not text:
        return fail("text 필요")
    try:
        import pyautogui
        # pyautogui.typewrite()는 ASCII 전용 — 한글/유니코드는 클립보드 경유
        try:
            import pyperclip
            pyperclip.copy(text)
            pyautogui.hotkey('ctrl', 'v')
        except Exception:
            pyautogui.typewrite(text, interval=interval)
        return ok(message=f"{len(text)}자 입력 완료")
    except Exception as e:
        return fail(str(e))


@app.post("/desktop/scroll")
def desktop_scroll(body: dict):
    x = body.get("x"); y = body.get("y"); amount = body.get("amount", 3)
    try:
        import pyautogui
        if x is not None and y is not None:
            pyautogui.moveTo(x, y)
        pyautogui.scroll(amount)
        return ok(message=f"스크롤: {amount}")
    except Exception as e:
        return fail(str(e))


@app.post("/desktop/drag")
def desktop_drag(body: dict):
    x1 = body.get("x1"); y1 = body.get("y1")
    x2 = body.get("x2"); y2 = body.get("y2")
    duration = body.get("duration", 0.5)
    if None in (x1, y1, x2, y2):
        return fail("x1, y1, x2, y2 필요")
    try:
        import pyautogui
        pyautogui.drag(x2-x1, y2-y1, duration=duration, button="left")
        return ok(message=f"드래그: ({x1},{y1}) → ({x2},{y2})")
    except Exception as e:
        return fail(str(e))


@app.post("/desktop/key")
def desktop_key(body: dict):
    keys = body.get("keys", [])
    if not keys:
        return fail("keys 필요")
    try:
        import pyautogui
        if isinstance(keys, list):
            pyautogui.hotkey(*keys)
        else:
            pyautogui.press(keys)
        return ok(message=f"키 입력: {keys}")
    except Exception as e:
        return fail(str(e))


@app.get("/desktop/screenshot")
@app.post("/desktop/screenshot")
def desktop_screenshot():
    try:
        import pyautogui
        from PIL import Image
        import io
        img = pyautogui.screenshot()
        buf = io.BytesIO()
        img.save(buf, format="PNG")
        b64 = base64.b64encode(buf.getvalue()).decode()
        return ok(image_base64=b64, width=img.width, height=img.height, message="스크린샷 완료")
    except Exception as e:
        return fail(str(e))


@app.get("/desktop/status")
def desktop_status():
    try:
        import pyautogui
        x, y = pyautogui.position()
        w, h = pyautogui.size()
        return ok(mouse_x=x, mouse_y=y, screen_width=w, screen_height=h,
                  busy=_desktop_task is not None and _desktop_task.is_alive())
    except Exception as e:
        return fail(str(e))


# ══════════════════════════════════════════════════════════════
#  Windows UI Automation (UIA) — 좌표가 아닌 '의미' 기반 데스크탑 자동화
#  ⚠️ pywinauto는 Windows 전용. import-guard로 Mac에서도 사이드카가 부팅된다.
#  ⚠️ [Windows QA 필요] 실제 클릭/입력 동작은 Windows 머신에서 검증해야 한다.
#     Go windowsAutomator가 /desktop/uia/*를 호출하고, RunSteps 닫힌 루프가
#     /desktop/uia/status(=Available()) 게이트를 통과해야만 실행된다.
# ══════════════════════════════════════════════════════════════
_uia_desktop = None
_uia_err = ""


def _uia():
    """pywinauto UIA Desktop 핸들 lazy 초기화 (실패 시 None + 사유 기록)."""
    global _uia_desktop, _uia_err
    if _uia_desktop is not None:
        return _uia_desktop
    try:
        import platform as _pf
        if _pf.system() != "Windows":
            _uia_err = "UIA는 Windows 전용"
            return None
        from pywinauto import Desktop  # Windows 전용 — import-guard로 보호
        _uia_desktop = Desktop(backend="uia")
        return _uia_desktop
    except Exception as e:
        _uia_err = f"pywinauto 미설치/초기화 실패: {e}"
        return None


def _find_in_top(top, sel: dict):
    """주어진 최상위 창에서 셀렉터로 후보 검색. returns 리스트(예외 시 [])."""
    try:
        kwargs = {}
        if sel.get("automation_id"):
            kwargs["auto_id"] = sel["automation_id"]
        if sel.get("role"):
            kwargs["control_type"] = sel["role"]
        cands = top.descendants(**kwargs) if kwargs else top.descendants()
        name = sel.get("name")
        if name:  # 접근성 이름 부분 일치
            cands = [c for c in cands if name in (c.window_text() or "")]
        return cands
    except Exception:
        return []


def _uia_find(sel: dict):
    """셀렉터(name/role/automation_id/index[/window])로 요소 검색.
       1) window 힌트가 있으면 제목 부분일치 창에서 우선 검색
       2) 활성(포그라운드) 창
       3) 그래도 없으면 모든 최상위 창 순회 (팝업/멀티윈도우 흐름 지원)
       returns (element|None, error_str)."""
    d = _uia()
    if d is None:
        return None, _uia_err or "UIA 미가용"
    try:
        idx = int(sel.get("index", 0) or 0)
        tried = []

        # 1) window 제목 힌트 우선
        win_hint = (sel.get("window") or "").strip()
        if win_hint:
            try:
                for w in d.windows():
                    try:
                        if win_hint in (w.window_text() or ""):
                            tried.append(w)
                            cands = _find_in_top(w, sel)
                            if cands:
                                return cands[idx if idx < len(cands) else 0], ""
                    except Exception:
                        continue
            except Exception:
                pass

        # 2) 활성 창
        try:
            top = d.window(active_only=True)
            cands = _find_in_top(top, sel)
            if cands:
                return cands[idx if idx < len(cands) else 0], ""
        except Exception:
            pass

        # 3) 전체 최상위 창 스캔 (마지막 수단 — 팝업이 포커스를 안 가진 경우)
        try:
            for w in d.windows():
                if w in tried:
                    continue
                cands = _find_in_top(w, sel)
                if cands:
                    return cands[idx if idx < len(cands) else 0], ""
        except Exception:
            pass

        return None, f"요소 없음: {sel}"
    except Exception as e:
        return None, str(e)


def _is_admin() -> bool:
    """관리자 권한 여부 — 대상 앱이 관리자인데 사이드카가 아니면 UIPI로 후킹/UIA가 막힌다."""
    try:
        import ctypes
        return bool(ctypes.windll.shell32.IsUserAnAdmin())
    except Exception:
        return False


@app.get("/desktop/uia/status")
def uia_status():
    import platform as _pf
    d = _uia()
    avail = d is not None
    admin = _is_admin()
    msg = "UIA 준비됨" if avail else "데스크탑 자동화는 Windows에서 사용 가능 (UIA 미가용)"
    if avail and not admin:
        msg += " · 일반 권한 (관리자 권한 앱은 제어 불가 — UIPI)"
    return ok(available=avail, platform=_pf.system().lower(), admin=admin,
              message=msg, detail=(_uia_err or None))


@app.post("/desktop/uia/find")
def uia_find(body: dict):
    el, err = _uia_find(body.get("selector", body))
    if el is None:
        return fail(err)
    try:
        return ok(found=True, name=el.window_text(), role=str(el.element_info.control_type))
    except Exception:
        return ok(found=True)


@app.post("/desktop/uia/click")
def uia_click(body: dict):
    el, err = _uia_find(body.get("selector", body))
    if el is None:
        return fail(err)
    try:
        el.click_input()
        return ok(message="클릭 완료")
    except Exception as e:
        return fail(str(e))


@app.post("/desktop/uia/set_text")
def uia_set_text(body: dict):
    text = body.get("text", body.get("value", ""))
    el, err = _uia_find(body.get("selector", body))
    if el is None:
        return fail(err)
    try:
        el.set_focus()
        # 한글/유니코드 안전: 클립보드 경유 (pyautogui.typewrite는 ASCII 전용)
        try:
            import pyperclip
            import pyautogui
            pyperclip.copy(text)
            pyautogui.hotkey('ctrl', 'a')
            pyautogui.hotkey('ctrl', 'v')
        except Exception:
            el.type_keys(text, with_spaces=True)
        return ok(message=f"{len(text)}자 입력 완료")
    except Exception as e:
        return fail(str(e))


@app.post("/desktop/uia/send_keys")
def uia_send_keys(body: dict):
    combo = body.get("keys", body.get("value", ""))
    if not combo:
        return fail("keys 필요")
    try:
        import pyautogui
        keys = [k.strip() for k in combo.replace('+', ' ').split() if k.strip()]
        if len(keys) > 1:
            pyautogui.hotkey(*keys)
        elif keys:
            pyautogui.press(keys[0])
        return ok(message=f"키 입력: {combo}")
    except Exception as e:
        return fail(str(e))


@app.post("/desktop/uia/verify")
def uia_verify(body: dict):
    sel = body.get("selector", {})
    expect = body.get("expect", "")
    el, err = _uia_find(sel)
    if el is None:
        return ok(verified=False, reason=err)
    try:
        txt = el.window_text() or ""
        verified = (expect in txt) if expect else True
        return ok(verified=verified, text=txt)
    except Exception as e:
        return ok(verified=False, reason=str(e))


@app.post("/desktop/uia/dclick")
def uia_dclick(body: dict):
    el, err = _uia_find(body.get("selector", body))
    if el is None:
        return fail(err)
    try:
        el.double_click_input()
        return ok(message="더블클릭 완료")
    except Exception as e:
        return fail(str(e))


@app.post("/desktop/uia/rclick")
def uia_rclick(body: dict):
    el, err = _uia_find(body.get("selector", body))
    if el is None:
        return fail(err)
    try:
        el.click_input(button="right")
        return ok(message="우클릭 완료")
    except Exception as e:
        return fail(str(e))


@app.post("/desktop/uia/scroll")
def uia_scroll(body: dict):
    amount = int(body.get("amount", 0) or 0)
    if amount == 0:
        return ok(message="스크롤 0 — no-op")
    sel = body.get("selector") or {}
    try:
        # 셀렉터가 있으면 해당 요소 위로 커서 이동 후 스크롤 (스크롤 컨테이너 타겟팅)
        if sel.get("name") or sel.get("role") or sel.get("automation_id"):
            el, err = _uia_find(sel)
            if el is not None:
                try:
                    r = el.rectangle()
                    import pyautogui
                    pyautogui.moveTo((r.left + r.right) // 2, (r.top + r.bottom) // 2)
                except Exception:
                    pass
        import pyautogui
        pyautogui.scroll(amount)
        return ok(message=f"스크롤 {amount}")
    except Exception as e:
        return fail(str(e))


# ──────────────────────────────────────────────────────────────
#  UIA 녹화기 (Recorder) — 사용자 클릭/입력을 AutoStep으로 자동 캡처
#  핵심: "한 번 시연하면 알아서 반복"의 '시연(녹화)' 부분.
#  pynput으로 마우스/키보드 이벤트를 잡고, 클릭 지점을 UIA ElementFromPoint로
#  '의미 셀렉터'(name/role/automation_id)로 변환한다. 텍스트 필드는 값 변화를
#  읽어 set_text, 그 외 클릭은 click 스텝으로 기록 → 기존 재생엔진으로 반복.
#  ⚠️ Windows 전용(UIA+pynput). Mac에서는 _uia()=None → start가 안전 거부.
# ──────────────────────────────────────────────────────────────
import threading as _threading
_rec_lock = _threading.Lock()
_rec_active = False
_rec_steps: List[dict] = []
_rec_pending = None          # (selector, UIAElementInfo, initial_value) — 입력 중인 필드
_rec_started_at = 0.0
_rec_last_ts = 0.0           # 마지막 사용자 행동 시각 — 간격을 wait 단계로 캡처
_rec_last_error = ""         # 캡처 실패 사유 (원격 디버깅용 — status로 노출)
_rec_skipped = 0             # 셀렉터 추출 실패로 버린 이벤트 수
_rec_mouse_listener = None
_rec_kb_listener = None

_REC_GAP_MIN_MS = 1200       # 이 이상 간격이면 wait 단계 삽입 (페이지 로딩 등)
_REC_GAP_MAX_MS = 8000       # wait 상한 (사용자 딴짓 과대 기록 방지)
_REC_DCLICK_SEC = 0.45       # 같은 요소 재클릭이 이 시간 내면 더블클릭으로 승격

_REC_CLICKABLE = ("Button", "Hyperlink", "MenuItem", "CheckBox",
                  "RadioButton", "ListItem", "TabItem", "SplitButton")
_REC_EDITABLE = ("Edit", "Document", "ComboBox")


def _uia_iface():
    try:
        from pywinauto.uia_defines import IUIA
        return IUIA().iuia
    except Exception:
        return None


def _element_from_point(x, y):
    """좌표 → UIA 요소. comtypes 환경별 POINT 타입이 달라 다중 폴백으로 방어.
       (B1: 실기에서 가장 먼저 깨질 지점 — 실패 사유를 _rec_last_error로 노출)"""
    global _rec_last_error
    iface = _uia_iface()
    if iface is None:
        _rec_last_error = "IUIA 인터페이스 없음"
        return None
    try:
        from pywinauto.uia_element_info import UIAElementInfo
    except Exception as e:
        _rec_last_error = f"UIAElementInfo import 실패: {e}"
        return None

    # POINT 타입 후보: comtypes 생성 tagPOINT → ctypes.wintypes → pywinauto
    candidates = []
    try:
        from comtypes.gen.UIAutomationClient import tagPOINT  # comtypes가 요구하는 정석 타입
        candidates.append(tagPOINT)
    except Exception:
        pass
    try:
        from ctypes import wintypes
        candidates.append(wintypes.POINT)
    except Exception:
        pass
    try:
        from pywinauto import win32structures
        candidates.append(win32structures.POINT)
    except Exception:
        pass

    last = None
    for P in candidates:
        try:
            pt = P()
            pt.x = int(x)
            pt.y = int(y)
            com = iface.ElementFromPoint(pt)
            if com is not None:
                return UIAElementInfo(com)
        except Exception as e:
            last = e
            continue
    _rec_last_error = f"ElementFromPoint 실패({len(candidates)}개 타입 시도): {last}"
    return None


def _elem_info_to_selector(info) -> dict:
    sel = {}
    for attr, key in (("name", "name"), ("control_type", "role"), ("automation_id", "automation_id")):
        try:
            v = getattr(info, attr, None)
            if v:
                sel[key] = v
        except Exception:
            pass
    return sel


def _normalize_target(info):
    """raw Text/Image 등을 클릭하면 의미 있는 조상(Button 등)으로 상승."""
    try:
        ct = info.control_type
    except Exception:
        return info
    if ct in _REC_CLICKABLE or ct in _REC_EDITABLE:
        return info
    cur = info
    for _ in range(4):
        try:
            p = cur.parent
        except Exception:
            break
        if p is None:
            break
        try:
            if p.control_type in _REC_CLICKABLE:
                return p
        except Exception:
            pass
        cur = p
    return info


def _read_value(info) -> str:
    """입력 필드 값 읽기 — ValuePattern → LegacyIAccessible → window_text 3단 폴백.
       (B2: 컨트롤 종류별로 지원 패턴이 달라 단일 경로는 잘 깨진다)"""
    try:
        from pywinauto.controls.uiawrapper import UIAWrapper
        w = UIAWrapper(info)
    except Exception:
        return ""
    try:  # 1) UIA ValuePattern
        v = w.iface_value.CurrentValue
        if v:
            return v
    except Exception:
        pass
    try:  # 2) Legacy IAccessible (옛 win32 컨트롤·일부 브라우저)
        v = (w.legacy_properties() or {}).get("Value")
        if v:
            return v
    except Exception:
        pass
    try:  # 3) 접근성 텍스트
        return w.window_text() or ""
    except Exception:
        return ""


def _top_level_of(info):
    """요소의 최상위 창(루트 바로 아래) element_info. 실패 시 None."""
    try:
        cur = info
        for _ in range(48):
            p = cur.parent
            if p is None:
                return cur  # cur == 데스크탑 루트 (이례적)
            if p.parent is None:
                return cur  # 부모가 루트 → cur가 최상위 창
            cur = p
    except Exception:
        pass
    return None


def _rt_id(info):
    try:
        rid = info.runtime_id
        return tuple(rid) if rid else None
    except Exception:
        return None


def _compute_index(info, sel: dict, top) -> int:
    """같은 셀렉터에 매칭되는 형제들 중 이 요소가 몇 번째인지 (C4: 동명 버튼 구분).
       runtime_id 비교로 자기 위치를 찾는다. 실패 시 0 (기존 동작과 동일)."""
    try:
        target = _rt_id(info)
        if target is None or top is None:
            return 0
        matches = []
        for d in top.descendants():
            try:
                if sel.get("role") and str(d.control_type) != sel["role"]:
                    continue
                if sel.get("automation_id") and d.automation_id != sel["automation_id"]:
                    continue
                if sel.get("name") and sel["name"] not in (d.name or ""):
                    continue
                matches.append(d)
            except Exception:
                continue
        for i, d in enumerate(matches):
            if _rt_id(d) == target:
                return i
        return 0
    except Exception:
        return 0


def _rec_flush_pending():
    """입력 중이던 필드의 현재 값을 읽어 set_text 스텝으로 확정 (lock 보유 상태에서 호출)."""
    global _rec_pending
    if _rec_pending is None:
        return
    sel, info, initial = _rec_pending
    _rec_pending = None
    cur = _read_value(info)
    if cur and cur != initial:
        _rec_steps.append({"kind": "set_text", "selector": sel, "value": cur})


def _rec_append(step: dict):
    """행동 간격이 크면 wait 단계를 먼저 삽입 후 step 기록 (C1: 페이지 로딩 시간 보존).
       lock 보유 상태에서 호출."""
    global _rec_last_ts
    now = time.time()
    gap_ms = int((now - _rec_last_ts) * 1000) if _rec_last_ts else 0
    if _rec_steps and gap_ms >= _REC_GAP_MIN_MS:
        _rec_steps.append({"kind": "wait", "value": str(min(gap_ms, _REC_GAP_MAX_MS))})
    _rec_steps.append(step)
    _rec_last_ts = now


def _rec_build_selector(x, y):
    """클릭 좌표 → (selector, info). 의미 상승 + index + window 힌트까지 채움.
       실패 시 (None, None) — 사유는 _rec_last_error/_rec_skipped에."""
    global _rec_skipped, _rec_last_error
    info = _element_from_point(x, y)
    if info is None:
        _rec_skipped += 1
        return None, None
    info = _normalize_target(info)
    sel = _elem_info_to_selector(info)
    if not sel:
        _rec_skipped += 1
        _rec_last_error = "요소에 name/role/automation_id 없음 (접근성 미지원 컨트롤)"
        return None, None
    top = _top_level_of(info)
    if top is not None:
        try:
            title = (top.name or "").strip()
            if title:
                sel["window"] = title[:60]  # C5: 멀티윈도우 재생 힌트
        except Exception:
            pass
    idx = _compute_index(info, sel, top)  # C4: 동명 요소 N번째
    if idx > 0:
        sel["index"] = idx
    return sel, info


def _rec_on_click(x, y, button, pressed):
    if not pressed:
        return
    btn = str(button).lower()
    with _rec_lock:
        if not _rec_active:
            return
        global _rec_last_ts
        _rec_flush_pending()
        sel, info = _rec_build_selector(x, y)
        if sel is None:
            _rec_last_ts = time.time()
            return
        if "right" in btn:  # C6: 우클릭
            _rec_append({"kind": "right_click", "selector": sel})
            return
        if "left" not in btn:
            _rec_last_ts = time.time()
            return
        role = sel.get("role", "")
        if role in _REC_EDITABLE:
            _rec_pending = (sel, info, _read_value(info))   # 입력 시작 — 값은 flush 때 확정
            _rec_last_ts = time.time()
            return
        # C6: 같은 요소를 짧은 간격으로 재클릭 → 더블클릭으로 승격
        now = time.time()
        if (_rec_steps and _rec_steps[-1].get("kind") == "click"
                and _rec_steps[-1].get("selector") == sel
                and (now - _rec_last_ts) <= _REC_DCLICK_SEC):
            _rec_steps[-1]["kind"] = "double_click"
            _rec_last_ts = now
            return
        _rec_append({"kind": "click", "selector": sel})


def _rec_on_scroll(x, y, dx, dy):
    """C6: 휠 스크롤 — 연속 스크롤은 직전 scroll 단계에 합산(coalesce)."""
    with _rec_lock:
        if not _rec_active or not dy:
            return
        global _rec_last_ts
        now = time.time()
        if (_rec_steps and _rec_steps[-1].get("kind") == "scroll"
                and (now - _rec_last_ts) <= 1.0):
            try:
                cur = int(_rec_steps[-1].get("value", "0"))
            except Exception:
                cur = 0
            _rec_steps[-1]["value"] = str(cur + int(dy))
            _rec_last_ts = now
            return
        _rec_append({"kind": "scroll", "value": str(int(dy))})


def _rec_on_press(key):
    with _rec_lock:
        if not _rec_active:
            return
        global _rec_last_ts
        _rec_last_ts = time.time()  # 타이핑도 '행동' — gap wait 과대 삽입 방지
        try:
            from pynput import keyboard as _kb
            if key in (_kb.Key.enter, _kb.Key.tab):
                _rec_flush_pending()
        except Exception:
            pass


@app.post("/desktop/uia/record/start")
def uia_record_start():
    global _rec_active, _rec_steps, _rec_pending, _rec_started_at
    global _rec_last_ts, _rec_last_error, _rec_skipped
    global _rec_mouse_listener, _rec_kb_listener
    if _uia() is None:
        return fail(_uia_err or "UIA 미가용 — 녹화는 Windows에서만 가능")
    try:
        from pynput import mouse, keyboard
    except Exception as e:
        return fail(f"pynput 미설치/초기화 실패: {e}")
    with _rec_lock:
        if _rec_active:
            return ok(recording=True, count=len(_rec_steps), message="이미 녹화 중")
        _rec_steps = []
        _rec_pending = None
        _rec_active = True
        _rec_started_at = time.time()
        _rec_last_ts = 0.0
        _rec_last_error = ""
        _rec_skipped = 0
    try:
        _rec_mouse_listener = mouse.Listener(on_click=_rec_on_click, on_scroll=_rec_on_scroll)
        _rec_mouse_listener.start()
        _rec_kb_listener = keyboard.Listener(on_press=_rec_on_press)
        _rec_kb_listener.start()
    except Exception as e:
        with _rec_lock:
            _rec_active = False
        return fail(f"입력 후킹 실패: {e}")
    msg = "녹화 시작 — 대상 앱에서 평소처럼 클릭/입력하세요"
    if not _is_admin():
        msg += " (일반 권한: 관리자 권한 앱은 캡처 안 됨)"
    return ok(recording=True, admin=_is_admin(), message=msg)


@app.get("/desktop/uia/record/status")
def uia_record_status():
    with _rec_lock:
        last = _rec_steps[-1] if _rec_steps else None
        elapsed = int(time.time() - _rec_started_at) if _rec_active else 0
        return ok(recording=_rec_active, count=len(_rec_steps), last=last, elapsed=elapsed,
                  skipped=_rec_skipped, last_error=(_rec_last_error or None))


@app.post("/desktop/uia/record/stop")
def uia_record_stop():
    global _rec_active, _rec_mouse_listener, _rec_kb_listener
    with _rec_lock:
        _rec_flush_pending()
        _rec_active = False
        steps = list(_rec_steps)
    for lst in (_rec_mouse_listener, _rec_kb_listener):
        try:
            if lst is not None:
                lst.stop()
        except Exception:
            pass
    _rec_mouse_listener = None
    _rec_kb_listener = None
    msg = f"녹화 완료 — {len(steps)}단계 캡처됨"
    if _rec_skipped:
        msg += f" (캡처 실패 {_rec_skipped}건)"
    return ok(recording=False, steps=steps, count=len(steps),
              skipped=_rec_skipped, last_error=(_rec_last_error or None), message=msg)


@app.post("/desktop/agent/run")
@app.post("/desktop-agent/run")
def desktop_agent_run(body: dict):
    task   = body.get("task", "")
    claude_key = body.get("claude_key", "") or CLAUDE_KEY
    if not task:
        return fail("task 필요")
    plan_prompt = f"""다음 작업을 수행하기 위한 단계별 컴퓨터 제어 액션을 JSON 배열로만 반환해줘.
사용 가능한 액션: click(x,y), type(text), key(keys), scroll(amount), wait(seconds)

작업: {task}

예시:
[{{"action":"click","x":100,"y":200}},{{"action":"type","text":"안녕하세요"}},{{"action":"key","keys":["ctrl","s"]}}]"""
    plan_str = groq_chat([{"role": "user", "content": plan_prompt}], max_tokens=600)
    try:
        actions = json.loads(re.search(r'\[.*\]', plan_str, re.DOTALL).group())
    except Exception:
        return fail("작업 계획 파싱 실패")
    results = []
    _desktop_cancel_flag.clear()
    import pyautogui, time as _time
    pyautogui.FAILSAFE = True
    for action in actions:
        if _desktop_cancel_flag.is_set():
            break
        a = action.get("action", "")
        try:
            if a == "click":
                pyautogui.click(action["x"], action["y"])
                results.append(f"클릭 ({action['x']},{action['y']})")
            elif a == "type":
                _t = action.get("text", "")
                try:
                    import pyperclip
                    pyperclip.copy(_t)
                    pyautogui.hotkey('ctrl', 'v')
                except Exception:
                    pyautogui.typewrite(_t, interval=0.03)
                results.append(f"입력: {_t[:20]}")
            elif a == "key":
                keys = action.get("keys", [])
                if isinstance(keys, list):
                    pyautogui.hotkey(*keys)
                else:
                    pyautogui.press(keys)
                results.append(f"키: {keys}")
            elif a == "scroll":
                pyautogui.scroll(action.get("amount", 3))
                results.append(f"스크롤 {action.get('amount', 3)}")
            elif a == "wait":
                _time.sleep(action.get("seconds", 1))
                results.append(f"대기 {action.get('seconds',1)}초")
            _time.sleep(0.3)
        except Exception as ex:
            results.append(f"오류: {ex}")
    return ok(task=task, actions_count=len(actions), results=results,
              message=f"작업 완료: {len(results)}개 액션 실행")


@app.post("/desktop/agent/cancel")
@app.post("/desktop-agent/cancel")
def desktop_agent_cancel():
    _desktop_cancel_flag.set()
    return ok(message="작업 취소됨")


@app.post("/desktop/approve")
def desktop_approve(body: dict):
    return ok(message="승인됨")


# ════════════════════════════════════════════════════════════
# 8단계 — Ollama
# ════════════════════════════════════════════════════════════

@app.get("/ollama/models")
def ollama_models(ollama_url: str = "http://localhost:11434"):
    try:
        r = requests.get(f"{ollama_url}/api/tags", timeout=5)
        models = [m["name"] for m in r.json().get("models", [])]
        return ok(models=models, count=len(models))
    except Exception as e:
        return ok(models=[], count=0, message=f"Ollama 연결 실패: {e}")


@app.post("/ollama/test")
def ollama_test(body: dict):
    ollama_url = body.get("ollama_url", "http://localhost:11434")
    model      = body.get("model", "llama3.2")
    try:
        r = requests.post(f"{ollama_url}/api/generate",
                          json={"model": model, "prompt": "hi", "stream": False},
                          timeout=15)
        return ok(model=model, response=r.json().get("response", ""),
                  message=f"Ollama {model} 연결 성공")
    except Exception as e:
        return fail(f"Ollama 연결 실패: {e}")


@app.post("/ollama/config")
def ollama_config(body: dict):
    return ok(enabled=True, url=body.get("url", "http://localhost:11434"),
              message="Ollama 설정 저장됨")


@app.post("/ollama/chat")
def ollama_chat(body: dict):
    ollama_url = body.get("ollama_url", "http://localhost:11434")
    model   = body.get("model", "llama3.2")
    message = body.get("message", "")
    if not message:
        return fail("message 필요")
    try:
        r = requests.post(f"{ollama_url}/api/generate",
                          json={"model": model, "prompt": message, "stream": False},
                          timeout=60)
        return ok(response=r.json().get("response", ""), model=model)
    except Exception as e:
        return fail(str(e))


# ════════════════════════════════════════════════════════════
# 9단계 — 워크플로우
# ════════════════════════════════════════════════════════════

WORKFLOW_TEMPLATES = [
    {"id": "morning_brief", "name": "모닝 브리핑", "description": "날씨+일정+이메일 요약",
     "steps": [{"action": "weather"}, {"action": "calendar_today"}, {"action": "email_inbox"}]},
    {"id": "daily_report", "name": "일일 보고서", "description": "PC 상태 + 보안 점검 후 이메일 발송",
     "steps": [{"action": "pc_status"}, {"action": "security_scan"}, {"action": "report_email"}]},
    {"id": "research_save", "name": "리서치 & 저장", "description": "검색 후 PDF 저장",
     "steps": [{"action": "deep_search"}, {"action": "search_pdf"}]},
    {"id": "meeting_prep", "name": "회의 준비", "description": "일정 확인 + 관련 문서 검색",
     "steps": [{"action": "calendar_today"}, {"action": "brain_search"}]},
    {"id": "file_cleanup", "name": "파일 정리", "description": "중복 파일 찾기 + 스마트 정리",
     "steps": [{"action": "file_duplicates"}, {"action": "smart_organize"}]},
]


@app.get("/workflow/templates")
def workflow_templates():
    return ok(templates=WORKFLOW_TEMPLATES, count=len(WORKFLOW_TEMPLATES))


@app.post("/workflow/from-text")
def workflow_from_text(body: dict):
    description = body.get("description", "")
    if not description:
        return fail("description 필요")
    prompt = f"""다음 설명을 워크플로우 YAML로 변환해줘. JSON 형식으로만 반환:
{{"name": "워크플로우명", "description": "설명", "steps": [{{"action": "액션명", "params": {{}}}}]}}

설명: {description}

가능한 액션: pc_status, security_scan, weather, calendar_today, email_inbox, deep_search,
file_search, brain_search, news_search, youtube_search, travel_time, virus_check"""
    result_str = groq_chat([{"role": "user", "content": prompt}], max_tokens=600)
    try:
        workflow = json.loads(re.search(r'\{.*\}', result_str, re.DOTALL).group())
    except Exception:
        workflow = {"name": description[:30], "description": description, "steps": []}
    workflow_id = int(time.time())
    con = sqlite3.connect(DB_PATH)
    con.execute("INSERT INTO workflows (name, description, yaml) VALUES (?,?,?)",
                (workflow.get("name",""), workflow.get("description",""), json.dumps(workflow)))
    con.commit(); con.close()
    return ok(workflow={**workflow, "id": workflow_id}, message="워크플로우 생성 완료")


@app.get("/workflow/list")
def workflow_list():
    con = sqlite3.connect(DB_PATH)
    rows = con.execute("SELECT id, name, description, created_at FROM workflows ORDER BY created_at DESC").fetchall()
    con.close()
    workflows = [{"id": r[0], "name": r[1], "description": r[2], "created_at": r[3]} for r in rows]
    return ok(workflows=workflows, count=len(workflows))


@app.post("/workflow/save")
def workflow_save(body: dict):
    name = body.get("name", ""); description = body.get("description", "")
    yaml_content = body.get("yaml", json.dumps(body))
    con = sqlite3.connect(DB_PATH)
    con.execute("INSERT INTO workflows (name, description, yaml) VALUES (?,?,?)",
                (name, description, yaml_content))
    con.commit(); con.close()
    return ok(message="워크플로우 저장 완료")


@app.delete("/workflow/delete")
def workflow_delete(body: dict):
    wf_id = body.get("id")
    if not wf_id:
        return fail("id 필요")
    con = sqlite3.connect(DB_PATH)
    con.execute("DELETE FROM workflows WHERE id=?", (wf_id,))
    con.commit(); con.close()
    return ok(message="삭제 완료")


@app.post("/workflow/run-now")
def workflow_run_now(body: dict):
    wf_id = body.get("id")
    con = sqlite3.connect(DB_PATH)
    row = con.execute("SELECT yaml FROM workflows WHERE id=?", (wf_id,)).fetchone()
    con.close()
    if not row:
        return fail("워크플로우 없음")
    wf = json.loads(row[0])
    return ok(workflow=wf, status="queued", message=f"워크플로우 '{wf.get('name','')}' 실행 대기")


# ════════════════════════════════════════════════════════════
# 10단계 — Multi-Agent
# ════════════════════════════════════════════════════════════

@app.post("/multi-agent/plan")
@app.post("/agent/multi/plan")
def multi_agent_plan(body: dict):
    task = body.get("task", "")
    if not task:
        return fail("task 필요")
    prompt = f"""다음 복잡한 작업을 병렬로 처리할 에이전트 팀 계획을 JSON으로만 반환해줘:
{{"agents": [{{"name": "에이전트명", "role": "역할", "action": "수행할 액션", "priority": 1}}], "summary": "전체 계획 요약"}}

작업: {task}"""
    result_str = groq_chat([{"role": "user", "content": prompt}], max_tokens=600)
    try:
        plan = json.loads(re.search(r'\{.*\}', result_str, re.DOTALL).group())
    except Exception:
        plan = {"agents": [{"name": "기본 에이전트", "role": "일반", "action": task, "priority": 1}],
                "summary": task}
    return ok(plan=plan, message=f"에이전트 팀 {len(plan.get('agents',[]))}명 배치 완료")


@app.post("/multi-agent/run")
@app.post("/agent/multi/run")
def multi_agent_run(body: dict):
    import uuid as _uuid
    task    = body.get("task", "")
    agents  = body.get("agents", [])
    if not task:
        return fail("task 필요")
    if not agents:
        plan_resp = multi_agent_plan(body)
        agents = plan_resp.get("plan", {}).get("agents", [])
        if not agents:
            agents = [{"name": "기본 에이전트", "role": "AI", "action": task}]
    results = []
    for agent in agents[:5]:
        agent_prompt = f"당신은 {agent.get('role','AI 에이전트')}입니다. 다음 작업을 수행해주세요: {agent.get('action', task)}"
        result = groq_chat([{"role": "user", "content": agent_prompt}], max_tokens=400)
        results.append({"agent": agent.get("name",""), "role": agent.get("role",""),
                        "result": result, "status": "done"})
    combined = groq_chat([
        {"role": "system", "content": "다음 여러 에이전트의 결과를 통합해서 최종 답변을 작성해줘."},
        {"role": "user", "content": json.dumps(results, ensure_ascii=False)}
    ], max_tokens=800)
    task_id = str(_uuid.uuid4())[:8]
    return ok(task=task, task_id=task_id, agents=results, combined_result=combined,
              message=combined or f"멀티 에이전트 {len(results)}명 완료")


@app.post("/multi-agent/stream/{task_id}")
@app.post("/agent/multi/stream")
def multi_agent_stream(body: dict, task_id: str = ""):
    return multi_agent_run(body)


@app.get("/multi-agent/agents")
@app.get("/agent/list")
def agent_list():
    agents = [
        {"id": "researcher", "name": "리서치 에이전트", "role": "웹 검색 및 정보 수집"},
        {"id": "analyst",    "name": "분석 에이전트",   "role": "데이터 분석 및 요약"},
        {"id": "writer",     "name": "작성 에이전트",   "role": "문서 작성 및 편집"},
        {"id": "executor",   "name": "실행 에이전트",   "role": "시스템 명령 실행"},
        {"id": "monitor",    "name": "모니터 에이전트", "role": "PC 상태 모니터링"},
    ]
    return ok(agents=agents, count=len(agents))


# ════════════════════════════════════════════════════════════
# 11단계 — 법률/의료/계약
# ════════════════════════════════════════════════════════════

def tavily_search_local(query: str, max_results: int = 5) -> list:
    # TAVILY_KEY 전역 변수 우선 (Go 백엔드가 주입), 그다음 환경변수
    tavily_key = TAVILY_KEY or os.environ.get("NEXUS_TAVILY_KEY", "")
    if not tavily_key:
        return []
    try:
        r = requests.post("https://api.tavily.com/search",
                          json={"api_key": tavily_key, "query": query,
                                "max_results": max_results, "search_depth": "advanced"},
                          timeout=15)
        return r.json().get("results", [])
    except Exception:
        return []


@app.post("/legal/search")
def legal_search(body: dict):
    query = body.get("query", "")
    if not query:
        return fail("query 필요")
    results = tavily_search_local(f"법률 판례 규정 {query}", 5)
    analysis = groq_chat([
        {"role": "system", "content": "당신은 법률 정보 전문가입니다. 검색 결과를 바탕으로 법률 정보를 제공하되 '전문 변호사 상담 권고' 문구를 포함하세요."},
        {"role": "user", "content": f"'{query}'에 대한 법률 정보: {json.dumps(results[:3], ensure_ascii=False)}"}
    ], max_tokens=800)
    return ok(results=results, analysis=analysis, query=query,
              disclaimer="법률 전문가 상담을 권고합니다.",
              message=f"법률 검색 '{query}' 완료")


@app.post("/medical/search")
def medical_search(body: dict):
    query = body.get("query", "")
    if not query:
        return fail("query 필요")
    results = tavily_search_local(f"의학 의료 증상 치료 {query}", 5)
    analysis = groq_chat([
        {"role": "system", "content": "당신은 의학 정보 전문가입니다. 검색 결과를 바탕으로 의료 정보를 제공하되 '의사 진료 권고' 문구를 포함하세요."},
        {"role": "user", "content": f"'{query}'에 대한 의료 정보: {json.dumps(results[:3], ensure_ascii=False)}"}
    ], max_tokens=800)
    return ok(results=results, analysis=analysis, query=query,
              disclaimer="의사 진료를 권고합니다.",
              message=f"의료 검색 '{query}' 완료")


@app.post("/contract/review")
def contract_review(body: dict):
    content = body.get("content", "")
    file_path = body.get("file_path", "")
    if file_path and os.path.exists(file_path):
        try:
            import fitz
            doc = fitz.open(file_path)
            content = "\n".join(page.get_text() for page in doc)
        except Exception:
            pass
    if not content:
        return fail("계약서 내용 또는 파일 경로 필요")
    analysis = groq_chat([
        {"role": "system", "content": "당신은 계약서 검토 전문가입니다. 다음 계약서를 분석하여 위험 조항, 누락 조항, 주의사항을 JSON으로만 반환하세요: {\"risk_clauses\":[],\"missing_clauses\":[],\"warnings\":[],\"summary\":\"\"}"},
        {"role": "user", "content": content[:4000]}
    ], max_tokens=1000)
    try:
        result = json.loads(re.search(r'\{.*\}', analysis, re.DOTALL).group())
    except Exception:
        result = {"risk_clauses": [], "missing_clauses": [], "warnings": [], "summary": analysis}
    return ok(**result, message="계약서 검토 완료", disclaimer="법률 전문가 최종 검토 권고")


# ════════════════════════════════════════════════════════════
# 12단계 — Task/Cron/Trigger (Go 연결 브릿지)
# ════════════════════════════════════════════════════════════

GO_BASE = "http://127.0.0.1:17891"

def proxy_to_go(path: str, body: dict = None, method: str = "POST"):
    try:
        if method == "GET":
            r = requests.get(f"{GO_BASE}{path}", timeout=10)
        else:
            r = requests.post(f"{GO_BASE}{path}", json=body or {}, timeout=10)
        return r.json()
    except Exception as e:
        return {"success": False, "message": str(e)}


@app.get("/tasks/list")
def task_list():
    return proxy_to_go("/api/tasks/list", method="GET")


@app.post("/tasks/cancel")
def task_cancel(body: dict):
    return proxy_to_go("/api/tasks/cancel", body)


@app.get("/triggers/list")
def trigger_list():
    return proxy_to_go("/api/triggers/list", method="GET")


@app.post("/triggers/add")
def trigger_add(body: dict):
    return proxy_to_go("/api/triggers/add", body)


@app.delete("/triggers/delete")
def trigger_delete(body: dict):
    return proxy_to_go("/api/triggers/delete", body)


@app.get("/cron/list")
def cron_list():
    return proxy_to_go("/api/cron/list", method="GET")


@app.post("/cron/add")
def cron_add(body: dict):
    return proxy_to_go("/api/cron/add", body)


@app.delete("/cron/delete")
def cron_delete(body: dict):
    return proxy_to_go("/api/cron/delete", body)


@app.post("/cron/run-now")
def cron_run_now(body: dict):
    return proxy_to_go("/api/cron/run-now", body)


# ════════════════════════════════════════════════════════════
# 헬스체크
# ════════════════════════════════════════════════════════════

@app.get("/health")
def health():
    return ok(service="nexus-python", port=17893, message="Python sidecar running")


if __name__ == "__main__":
    groq_key = ""
    for arg in sys.argv[1:]:
        if arg.startswith("--groq-key="):
            groq_key = arg.split("=", 1)[1]
        elif arg.startswith("--claude-key="):
            os.environ["NEXUS_CLAUDE_KEY"] = arg.split("=", 1)[1]
        elif arg.startswith("--tavily-key="):
            os.environ["NEXUS_TAVILY_KEY"] = arg.split("=", 1)[1]
    if groq_key:
        GROQ_KEY = groq_key
    # 바인드 주소: 기본은 127.0.0.1(보안 — localhost 전용).
    # ⚠️ 원격 테스트(예: Mac→Parallels VM)로 노출하려면 NEXUS_SIDECAR_HOST=0.0.0.0 설정.
    #    데스크탑 제어(pyautogui/UIA)를 네트워크에 여는 것이므로 신뢰된 망에서만 사용하고
    #    테스트 후 해제할 것. (방화벽 인바운드 17893 허용 필요)
    host = os.environ.get("NEXUS_SIDECAR_HOST", "127.0.0.1")
    uvicorn.run(app, host=host, port=17893, log_level="error")
