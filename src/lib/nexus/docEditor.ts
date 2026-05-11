const BASE = 'http://127.0.0.1:17891'

export interface UploadResult {
  success: boolean
  file_path: string
  filename: string
  ext: string
  size: number
  preview?: {
    sheets?: string[]
    rows?: string[][]
    total_rows?: number
    text?: string
  }
  preview_error?: string
  message?: string
}

export interface EditResult {
  success: boolean
  out_path: string
  summary: string
  message: string
  operations_count?: number
  operations?: string[]
}

/** 파일을 백엔드에 업로드하고 임시 경로 + 미리보기를 반환 */
export async function uploadDocFile(file: File): Promise<UploadResult> {
  const form = new FormData()
  form.append('file', file)
  const res = await fetch(`${BASE}/api/docs/upload`, { method: 'POST', body: form })
  if (!res.ok) throw new Error(`업로드 실패: ${res.status}`)
  return res.json()
}

/** AI에게 문서 편집 지시 → 수정된 파일 바탕화면 저장 */
export async function aiEditDoc(
  filePath: string,
  instruction: string,
  sheetName?: string,
  saveAs?: string,
): Promise<EditResult> {
  const res = await fetch(`${BASE}/api/docs/ai-edit`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ file_path: filePath, instruction, sheet_name: sheetName ?? '', save_as: saveAs ?? '' }),
  })
  if (!res.ok) {
    const err = await res.json().catch(() => ({}))
    throw new Error((err as any).message ?? `편집 실패: ${res.status}`)
  }
  return res.json()
}

/** Excel 파일 내용 읽기 → 2D 배열 */
export async function readExcel(filePath: string, sheet?: string) {
  const params = new URLSearchParams({ path: filePath, sheet: sheet ?? '' })
  const res = await fetch(`${BASE}/api/excel/read?${params}`)
  if (!res.ok) throw new Error(`읽기 실패: ${res.status}`)
  return res.json() as Promise<{ success: boolean; sheets: string[]; data: string[][]; rows: number }>
}

/** 파일 확장자로 문서 타입 판별 */
export function docTypeLabel(ext: string): string {
  const map: Record<string, string> = {
    '.xlsx': 'Excel',
    '.xls': 'Excel',
    '.xlsm': 'Excel(매크로)',
    '.docx': 'Word',
    '.doc': 'Word',
    '.csv': 'CSV',
    '.txt': '텍스트',
    '.md': 'Markdown',
    '.pdf': 'PDF',
  }
  return map[ext.toLowerCase()] ?? '문서'
}

/** 파일이 Excel 계열인지 */
export function isExcelFile(ext: string): boolean {
  return ['.xlsx', '.xls', '.xlsm'].includes(ext.toLowerCase())
}
