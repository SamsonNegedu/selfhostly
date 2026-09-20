import { cn } from '@/shared/lib/utils'
import type { StatusKind } from '@/shared/lib/status'

const STROKE_CLASSES: Record<StatusKind, string> = {
    ok: 'stroke-status-ok',
    warn: 'stroke-status-warn',
    err: 'stroke-status-err',
    info: 'stroke-status-info',
    idle: 'stroke-status-idle',
}

const FILL_CLASSES: Record<StatusKind, string> = {
    ok: 'fill-status-ok',
    warn: 'fill-status-warn',
    err: 'fill-status-err',
    info: 'fill-status-info',
    idle: 'fill-status-idle',
}

const AREA_OPACITY = 0.12
const GRID_LINES = [25, 50, 75]
const CHART_SIZE = 100
const SPARK_PADDING = 2

interface ChartBaseProps {
    data: number[]
    tone?: StatusKind
    label?: string
    className?: string
}

// Maps a value to a y coordinate. A missing range collapses to a flat line in the middle.
function scale(value: number, min: number, max: number, height: number, padding: number) {
    const range = max - min
    if (range === 0) return height / 2
    return height - padding - ((value - min) / range) * (height - padding * 2)
}

interface SparklineProps extends ChartBaseProps {
    width?: number
    height?: number
}

// A tiny trend line with no axes. Decorative unless a label is given.
function Sparkline({ data, width = 96, height = 28, tone = 'info', label, className }: SparklineProps) {
    if (data.length < 2) return null
    const min = Math.min(...data)
    const max = Math.max(...data)
    const points = data
        .map(
            (value, index) => `${(index / (data.length - 1)) * width},${scale(value, min, max, height, SPARK_PADDING)}`,
        )
        .join(' ')

    return (
        <svg
            width={width}
            height={height}
            viewBox={`0 0 ${width} ${height}`}
            role={label ? 'img' : undefined}
            aria-label={label}
            aria-hidden={label ? undefined : true}
            className={cn('shrink-0', className)}
        >
            <polyline
                points={points}
                fill="none"
                strokeWidth={1.75}
                strokeLinejoin="round"
                strokeLinecap="round"
                className={STROKE_CLASSES[tone]}
            />
        </svg>
    )
}

interface AreaChartProps extends ChartBaseProps {
    height?: number
    max?: number
}

// A filled trend over a fixed 0 to max scale, stretched to the width of its container.
function AreaChart({ data, height = 90, tone = 'info', label, max = CHART_SIZE, className }: AreaChartProps) {
    if (data.length < 2) return null
    const y = (value: number) => CHART_SIZE - (Math.min(max, Math.max(0, value)) / max) * CHART_SIZE
    const line = data.map((value, index) => `${(index / (data.length - 1)) * CHART_SIZE},${y(value)}`).join(' ')

    return (
        <svg
            viewBox={`0 0 ${CHART_SIZE} ${CHART_SIZE}`}
            preserveAspectRatio="none"
            width="100%"
            height={height}
            role={label ? 'img' : undefined}
            aria-label={label}
            aria-hidden={label ? undefined : true}
            className={cn('block', className)}
        >
            {GRID_LINES.map((grid) => (
                <line
                    key={grid}
                    x1={0}
                    x2={CHART_SIZE}
                    y1={y(grid)}
                    y2={y(grid)}
                    strokeWidth={1}
                    vectorEffect="non-scaling-stroke"
                    className="stroke-border"
                />
            ))}
            <polygon
                points={`0,${CHART_SIZE} ${line} ${CHART_SIZE},${CHART_SIZE}`}
                fillOpacity={AREA_OPACITY}
                className={FILL_CLASSES[tone]}
            />
            <polyline
                points={line}
                fill="none"
                strokeWidth={2}
                strokeLinejoin="round"
                vectorEffect="non-scaling-stroke"
                className={STROKE_CLASSES[tone]}
            />
        </svg>
    )
}

export { Sparkline, AreaChart }
