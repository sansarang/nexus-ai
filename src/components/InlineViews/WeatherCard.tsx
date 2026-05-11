interface WeatherData { city?: string; temp?: number; condition?: string; humidity?: number; pm25?: number }

const WEATHER_ICONS: Record<string, string> = {
  맑음: '☀️', 흐림: '☁️', 비: '🌧️', 눈: '❄️', 구름: '⛅', 안개: '🌫️',
}

export function WeatherCard({ data }: { data: unknown }) {
  const d = (data ?? {}) as WeatherData

  return (
    <div style={{ padding: 16 }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 16, marginBottom: 12 }}>
        <div style={{ fontSize: 48 }}>
          {WEATHER_ICONS[d.condition ?? ''] ?? '🌤️'}
        </div>
        <div>
          <div style={{ fontSize: 13, color: 'var(--text-secondary)' }}>{d.city ?? '서울'}</div>
          <div style={{ fontSize: 36, fontWeight: 800, color: 'var(--text-primary)' }}>
            {d.temp ?? '--'}°C
          </div>
          <div style={{ fontSize: 13, color: 'var(--text-secondary)' }}>{d.condition ?? '정보 없음'}</div>
        </div>
      </div>

      <div style={{ display: 'flex', gap: 12 }}>
        {d.humidity != null && (
          <div style={{ padding: '6px 12px', borderRadius: 8, background: 'var(--bg-elevated)', fontSize: 12, color: 'var(--text-secondary)' }}>
            💧 습도 {d.humidity}%
          </div>
        )}
        {d.pm25 != null && (
          <div style={{
            padding: '6px 12px', borderRadius: 8, background: 'var(--bg-elevated)', fontSize: 12,
            color: d.pm25 <= 15 ? '#22c55e' : d.pm25 <= 35 ? '#f59e0b' : '#ef4444',
          }}>
            🌫️ PM2.5 {d.pm25} µg/m³
          </div>
        )}
      </div>
    </div>
  )
}
