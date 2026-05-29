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

@app.post("/youtube/search")
def youtube_search(body: dict):
    query = body.get("query", "")
    max_items = body.get("max_items", 10)
    if not query:
        return fail("query 필요")
    try:
        import yt_dlp
        ydl_opts = {
            "quiet": True, "no_warnings": True,
            "extract_flat": True, "skip_download": True,
        }
        results = []
        with yt_dlp.YoutubeDL(ydl_opts) as ydl:
            info = ydl.extract_info(f"ytsearch{max_items}:{query}", download=False)
            for e in (info.get("entries") or []):
                if e:
                    results.append({
                        "title":    e.get("title", ""),
                        "url":      e.get("webpage_url") or f"https://www.youtube.com/watch?v={e.get('id','')}",
                        "channel":  e.get("channel") or e.get("uploader", ""),
                        "duration": str(e.get("duration", "")),
                        "views":    str(e.get("view_count", "")),
                    })
        return ok(items=results, count=len(results),
                  message=f"YouTube '{query}' 검색 결과 {len(results)}개",
                  summary=f"유튜브에서 '{query}' 관련 영상 {len(results)}개를 찾았어요!")
    except Exception as e:
        return fail(str(e))


@app.get("/tiktok/trending")
def tiktok_trending():
    """TikTok 트렌딩 — yt-dlp ttsearch fallback"""
    try:
        import yt_dlp
        ydl_opts = {"quiet": True, "no_warnings": True, "extract_flat": True, "skip_download": True}
        results = []
        with yt_dlp.YoutubeDL(ydl_opts) as ydl:
            info = ydl.extract_info("ttsearch15:trending korea viral 2025", download=False)
            for e in (info.get("entries") or []):
                if e:
                    results.append({
                        "title":   e.get("title", ""),
                        "url":     e.get("webpage_url", ""),
                        "author":  e.get("uploader", ""),
                        "views":   str(e.get("view_count", "")),
                    })
        return ok(items=results, count=len(results), source="yt_dlp",
                  message=f"🔥 TikTok 트렌딩 {len(results)}개")
    except Exception as e:
        return fail(str(e))


@app.post("/tiktok/profile")
def tiktok_profile(body: dict):
    """특정 TikTok 계정의 최근 영상 — yt-dlp"""
    username = body.get("username", "").lstrip("@")
    limit = body.get("limit", 10)
    if not username:
        return fail("username 필요")
    try:
        import yt_dlp
        url = f"https://www.tiktok.com/@{username}"
        ydl_opts = {
            "quiet": True, "no_warnings": True,
            "extract_flat": True, "skip_download": True,
            "playlistend": limit,
        }
        results = []
        with yt_dlp.YoutubeDL(ydl_opts) as ydl:
            info = ydl.extract_info(url, download=False)
            for e in (info.get("entries") or []):
                if e:
                    results.append({
                        "title":  e.get("title", ""),
                        "url":    e.get("webpage_url", ""),
                        "author": username,
                        "views":  str(e.get("view_count", "")),
                    })
        return ok(items=results, count=len(results),
                  message=f"👤 @{username} 최근 영상 {len(results)}개")
    except Exception as e:
        return fail(str(e))


@app.post("/tiktok/search")
def tiktok_search(body: dict):
    query = body.get("query", "")
    max_items = body.get("max_items", 10)
    if not query:
        return fail("query 필요")
    try:
        import yt_dlp
        ydl_opts = {"quiet": True, "no_warnings": True, "extract_flat": True, "skip_download": True}
        results = []
        with yt_dlp.YoutubeDL(ydl_opts) as ydl:
            info = ydl.extract_info(f"ttsearch{max_items}:{query}", download=False)
            for e in (info.get("entries") or []):
                if e:
                    results.append({
                        "title":   e.get("title", ""),
                        "url":     e.get("webpage_url", ""),
                        "channel": e.get("uploader", ""),
                        "views":   str(e.get("view_count", "")),
                    })
        return ok(items=results, count=len(results),
                  message=f"TikTok '{query}' 검색 결과 {len(results)}개",
                  summary=f"틱톡에서 '{query}' 관련 영상 {len(results)}개를 찾았어요!")
    except Exception as e:
        return fail(str(e))


@app.post("/ytmusic/search")
def ytmusic_search(body: dict):
    query = body.get("query", "")
    limit = body.get("limit", 10)
    if not query:
        return fail("query 필요")
    try:
        from ytmusicapi import YTMusic
        yt = YTMusic()
        raw = yt.search(query, filter="songs", limit=limit)
        items = []
        for r in raw:
            items.append({
                "title":   r.get("title", ""),
                "artist":  ", ".join(a["name"] for a in r.get("artists", [])),
                "album":   (r.get("album") or {}).get("name", ""),
                "url":     f"https://music.youtube.com/watch?v={r.get('videoId','')}",
                "thumbnail": ((r.get("thumbnails") or [{}])[-1]).get("url", ""),
            })
        return ok(items=items, count=len(items), message=f"YouTube Music '{query}' {len(items)}개")
    except Exception as e:
        return fail(str(e))


@app.post("/video/search-enhanced")
def video_search_enhanced(body: dict):
    query = body.get("query", "")
    platform = body.get("platform", "youtube")
    max_items = body.get("max_items", 10)
    if not query:
        return fail("query 필요")
    search_prefix = {"youtube": "ytsearch", "tiktok": "ttsearch"}.get(platform, "ytsearch")
    try:
        import yt_dlp
        ydl_opts = {"quiet": True, "no_warnings": True, "extract_flat": True}
        results = []
        with yt_dlp.YoutubeDL(ydl_opts) as ydl:
            info = ydl.extract_info(f"{search_prefix}{max_items}:{query}", download=False)
            for e in (info.get("entries") or []):
                if e:
                    results.append({
                        "title": e.get("title", ""), "url": e.get("webpage_url", ""),
                        "channel": e.get("uploader", ""), "duration": str(e.get("duration", "")),
                        "views": str(e.get("view_count", "")), "upload_date": e.get("upload_date", ""),
                    })
        return ok(items=results, count=len(results))
    except Exception as e:
        return fail(str(e))


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
    claude_key = body.get("claude_key", os.environ.get("NEXUS_CLAUDE_KEY", ""))
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


@app.post("/screenshot/translate")
def screenshot_translate(body: dict):
    image_base64 = body.get("image_base64", "")
    target_lang = body.get("target_lang", "ko")
    claude_key = body.get("claude_key", os.environ.get("NEXUS_CLAUDE_KEY", ""))
    if not image_base64 or not claude_key:
        return fail("image_base64, claude_key 필요")
    try:
        prompt = f"이 이미지의 모든 텍스트를 {'한국어' if target_lang=='ko' else '영어'}로 번역해줘. 원본 레이아웃 구조를 유지해."
        r = requests.post(
            "https://api.anthropic.com/v1/messages",
            headers={"x-api-key": claude_key, "anthropic-version": "2023-06-01",
                     "Content-Type": "application/json"},
            json={"model": "claude-haiku-4-5-20251001", "max_tokens": 2048,
                  "messages": [{"role": "user", "content": [
                      {"type": "image", "source": {"type": "base64", "media_type": "image/png", "data": image_base64}},
                      {"type": "text", "text": prompt}
                  ]}]},
            timeout=30
        )
        text = r.json()["content"][0]["text"]
        return ok(translated=text, message="번역 완료")
    except Exception as e:
        return fail(str(e))


# ════════════════════════════════════════════════════════════
# 4단계 — Email AI
# ════════════════════════════════════════════════════════════

@app.post("/email/classify")
def email_classify(body: dict):
    emails = body.get("emails", [])
    if not emails:
        return fail("emails 필요")
    classified = []
    for email in emails:
        subject = email.get("subject", "")
        sender  = email.get("from", "")
        preview = email.get("preview", "")[:300]
        prompt = f"""다음 이메일을 분류해줘. JSON만 반환:
{{"category": "업무|광고|뉴스레터|개인|청구서|알림|기타", "priority": "높음|보통|낮음", "summary": "한줄요약", "action": "답장필요|확인만|삭제가능"}}

제목: {subject}
발신: {sender}
내용: {preview}"""
        result_str = groq_chat([{"role": "user", "content": prompt}], max_tokens=200)
        try:
            result = json.loads(re.search(r'\{.*\}', result_str, re.DOTALL).group())
        except Exception:
            result = {"category": "기타", "priority": "보통", "summary": subject, "action": "확인만"}
        classified.append({**email, **result})
    return ok(emails=classified, count=len(classified), message=f"이메일 {len(classified)}개 분류 완료")


@app.post("/email/draft-reply")
def email_draft_reply(body: dict):
    original = body.get("original", "")
    context  = body.get("context", "")
    tone     = body.get("tone", "professional")
    lang     = body.get("lang", "ko")
    if not original:
        return fail("original 이메일 내용 필요")
    tone_map = {"professional": "전문적이고 공손한", "casual": "친근한", "formal": "격식있는"}
    tone_str = tone_map.get(tone, "전문적이고 공손한")
    prompt = f"""{tone_str} 어투로 다음 이메일에 대한 답장 초안을 작성해줘.
추가 맥락: {context}

원본 이메일:
{original}

답장 초안만 작성해줘."""
    draft = groq_chat([{"role": "user", "content": prompt}], max_tokens=800)
    return ok(draft=draft, message="답장 초안 생성 완료")


@app.post("/email/extract-events")
def email_extract_events(body: dict):
    content = body.get("content", "")
    if not content:
        return fail("content 필요")
    prompt = f"""다음 이메일에서 일정/이벤트 정보를 추출해줘. JSON 배열만 반환:
[{{"title": "제목", "date": "날짜", "time": "시간", "location": "장소", "description": "설명"}}]

이메일:
{content[:2000]}"""
    result_str = groq_chat([{"role": "user", "content": prompt}], max_tokens=500)
    try:
        events = json.loads(re.search(r'\[.*\]', result_str, re.DOTALL).group())
    except Exception:
        events = []
    return ok(events=events, count=len(events), message=f"일정 {len(events)}개 추출")


@app.post("/calendar/find-slot")
def calendar_find_slot(body: dict):
    duration_min = body.get("duration_min", 60)
    prefer_time  = body.get("prefer_time", "morning")
    within_days  = body.get("within_days", 7)
    from datetime import datetime, timedelta
    now = datetime.now()
    prefer_hour = {"morning": 9, "afternoon": 14, "evening": 17}.get(prefer_time, 9)
    slots = []
    for day in range(1, within_days + 1):
        if len(slots) >= 5:
            break
        date = now + timedelta(days=day)
        if date.weekday() >= 5:
            continue
        end_hour = prefer_hour + duration_min // 60
        slots.append({
            "date":       date.strftime("%Y-%m-%d"),
            "start_time": f"{prefer_hour:02d}:00",
            "end_time":   f"{end_hour:02d}:{duration_min%60:02d}",
        })
    return ok(slots=slots, message=f"{duration_min}분 미팅 가능 시간대 {len(slots)}개")


@app.post("/calendar/smart-add")
def calendar_smart_add(body: dict):
    text = body.get("text", "")
    if not text:
        return fail("text 필요")
    prompt = f"""다음 자연어 문장에서 일정 정보를 추출해줘. JSON만 반환:
{{"title":"제목","date":"YYYY-MM-DD","time":"HH:MM","duration_min":60,"location":"장소"}}
문장: {text}"""
    result_str = groq_chat([{"role": "user", "content": prompt}], max_tokens=200)
    try:
        event = json.loads(re.search(r'\{.*\}', result_str, re.DOTALL).group())
    except Exception:
        return fail("일정 파싱 실패")
    return ok(event=event, message=f"일정 추출 완료: {event.get('title','')}")


@app.post("/content/script")
def content_script(body: dict):
    topic    = body.get("topic", "")
    platform = body.get("platform", "youtube")
    duration = body.get("duration", "3분")
    style    = body.get("style", "informative")
    if not topic:
        return fail("topic 필요")
    prompt = f"""다음 주제로 {platform} {duration} 영상 스크립트를 {style} 스타일로 작성해줘.
주제: {topic}
형식: 인트로 → 본론 → 아웃트로 구조. 한국어로 작성."""
    script = groq_chat([{"role": "user", "content": prompt}], max_tokens=1500)
    return ok(script=script, topic=topic, platform=platform,
              message=f"'{topic}' {platform} 스크립트 생성 완료")


@app.post("/imap/classify")
def imap_classify(body: dict):
    return email_classify(body)


@app.post("/imap/reply-suggestions")
def imap_reply_suggestions(body: dict):
    email_content = body.get("content", "")
    prompt = f"""다음 이메일에 대한 간단한 답장 3가지를 제안해줘. JSON 배열만:
[{{"label": "버튼 텍스트", "text": "답장 내용"}}]

이메일: {email_content[:1000]}"""
    result_str = groq_chat([{"role": "user", "content": prompt}], max_tokens=400)
    try:
        suggestions = json.loads(re.search(r'\[.*\]', result_str, re.DOTALL).group())
    except Exception:
        suggestions = [{"label": "확인했습니다", "text": "네, 확인했습니다. 감사합니다."}]
    return ok(suggestions=suggestions)


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

@app.post("/stock/quote")
def stock_quote(body: dict):
    symbol = body.get("symbol", "").upper()
    if not symbol:
        return fail("symbol 필요")
    try:
        import yfinance as yf
        t = yf.Ticker(symbol)
        info = t.info
        hist = t.history(period="5d")
        current = float(hist["Close"].iloc[-1]) if len(hist) else 0
        prev    = float(hist["Close"].iloc[-2]) if len(hist) > 1 else current
        change  = current - prev
        pct     = (change / prev * 100) if prev else 0
        return ok(symbol=symbol, price=round(current, 2),
                  change=round(change, 2), change_pct=round(pct, 2),
                  currency=info.get("currency", "USD"),
                  company=info.get("longName", symbol),
                  market_cap=info.get("marketCap", 0),
                  message=f"{symbol} ${current:.2f} ({pct:+.2f}%)")
    except Exception as e:
        return fail(str(e))


@app.post("/stock/analysis")
def stock_analysis(body: dict):
    symbol = body.get("symbol", "").upper()
    period = body.get("period", "3mo")
    if not symbol:
        return fail("symbol 필요")
    try:
        import yfinance as yf, pandas as pd
        t = yf.Ticker(symbol)
        hist = t.history(period=period)
        if hist.empty:
            return fail("데이터 없음")
        hist["MA20"] = hist["Close"].rolling(20).mean()
        hist["MA60"] = hist["Close"].rolling(60).mean()
        hist["RSI"]  = 100 - (100 / (1 + hist["Close"].diff().clip(lower=0).rolling(14).mean() /
                                         (-hist["Close"].diff().clip(upper=0)).rolling(14).mean()))
        latest = hist.iloc[-1]
        trend = "상승" if latest["MA20"] > latest["MA60"] else "하락"
        rsi_signal = "과매수" if latest["RSI"] > 70 else ("과매도" if latest["RSI"] < 30 else "중립")
        return ok(symbol=symbol, period=period,
                  high=round(float(hist["High"].max()), 2),
                  low=round(float(hist["Low"].min()), 2),
                  current=round(float(latest["Close"]), 2),
                  ma20=round(float(latest["MA20"]) if pd.notna(latest["MA20"]) else 0, 2),
                  ma60=round(float(latest["MA60"]) if pd.notna(latest["MA60"]) else 0, 2),
                  rsi=round(float(latest["RSI"]) if pd.notna(latest["RSI"]) else 0, 1),
                  trend=trend, rsi_signal=rsi_signal,
                  message=f"{symbol} {period} 분석: {trend} 추세, RSI {rsi_signal}")
    except Exception as e:
        return fail(str(e))


@app.get("/stock/watchlist")
def stock_watchlist_get():
    con = sqlite3.connect(DB_PATH)
    rows = con.execute("SELECT symbol, name, added_at FROM stock_watchlist").fetchall()
    con.close()
    return ok(watchlist=[{"symbol": r[0], "name": r[1], "added_at": r[2]} for r in rows])


@app.post("/stock/watchlist/add")
def stock_watchlist_add(body: dict):
    symbol = body.get("symbol", "").upper()
    name   = body.get("name", symbol)
    if not symbol:
        return fail("symbol 필요")
    con = sqlite3.connect(DB_PATH)
    try:
        con.execute("INSERT OR IGNORE INTO stock_watchlist (symbol, name) VALUES (?, ?)", (symbol, name))
        con.commit()
    finally:
        con.close()
    return ok(message=f"{symbol} 관심목록 추가")


@app.delete("/stock/watchlist/delete")
def stock_watchlist_delete(body: dict):
    symbol = body.get("symbol", "").upper()
    con = sqlite3.connect(DB_PATH)
    con.execute("DELETE FROM stock_watchlist WHERE symbol=?", (symbol,))
    con.commit(); con.close()
    return ok(message=f"{symbol} 삭제됨")


@app.post("/security/shodan")
def shodan_audit(body: dict):
    api_key = body.get("api_key", os.environ.get("NEXUS_SHODAN_KEY", ""))
    ip      = body.get("ip", "")
    if not api_key:
        return fail("Shodan API 키 필요")
    try:
        import shodan as shodan_lib
        api = shodan_lib.Shodan(api_key)
        if ip:
            host = api.host(ip)
            return ok(ip=ip, hostnames=host.get("hostnames", []),
                      ports=host.get("ports", []), vulns=list(host.get("vulns", [])),
                      org=host.get("org", ""), country=host.get("country_name", ""),
                      message=f"{ip} Shodan 분석 완료")
        else:
            myip = api.tools.myip()
            return ok(my_ip=myip, message=f"내 IP: {myip}")
    except Exception as e:
        return fail(str(e))


@app.get("/wayback/available")
def wayback_available(url: str = ""):
    if not url:
        return fail("url 필요")
    try:
        from waybackpy import WaybackMachineSaveAPI, WaybackMachineAvailabilityAPI
        avail = WaybackMachineAvailabilityAPI(url)
        archive = avail.newest()
        return ok(available=True, archive_url=archive.archive_url,
                  timestamp=str(archive.timestamp), message="Wayback Machine 아카이브 있음")
    except Exception as e:
        return ok(available=False, message=str(e))


@app.get("/wayback/snapshots")
def wayback_snapshots(url: str = "", limit: int = 10):
    if not url:
        return fail("url 필요")
    try:
        from waybackpy import WaybackMachineCDXServerAPI
        cdx = WaybackMachineCDXServerAPI(url, user_agent="NexusBot/1.0")
        snapshots = []
        for s in cdx.snapshots():
            snapshots.append({"timestamp": str(s.timestamp), "archive_url": s.archive_url,
                               "status": s.statuscode})
            if len(snapshots) >= limit:
                break
        return ok(snapshots=snapshots, count=len(snapshots))
    except Exception as e:
        return fail(str(e))


@app.post("/search/anonymous")
def anonymous_search(body: dict):
    query = body.get("query", "")
    engine = body.get("engine", "ddg")
    if not query:
        return fail("query 필요")
    try:
        headers = {"User-Agent": "Mozilla/5.0"}
        if engine == "ddg":
            r = requests.get("https://api.duckduckgo.com/",
                             params={"q": query, "format": "json", "no_html": 1},
                             headers=headers, timeout=10)
            data = r.json()
            results = [{"title": t.get("Text", ""), "url": t.get("FirstURL", "")}
                       for t in data.get("RelatedTopics", [])[:10] if t.get("FirstURL")]
            return ok(results=results, count=len(results), source="DuckDuckGo")
        return fail("지원하지 않는 엔진")
    except Exception as e:
        return fail(str(e))


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


@app.post("/desktop/agent/run")
@app.post("/desktop-agent/run")
def desktop_agent_run(body: dict):
    task   = body.get("task", "")
    claude_key = body.get("claude_key", os.environ.get("NEXUS_CLAUDE_KEY", ""))
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
                pyautogui.typewrite(action.get("text",""), interval=0.03)
                results.append(f"입력: {action.get('text','')[:20]}")
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
    uvicorn.run(app, host="127.0.0.1", port=17893, log_level="error")
