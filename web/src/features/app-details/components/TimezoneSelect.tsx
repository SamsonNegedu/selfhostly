import { timeZones } from '../lib/schedule-cron'

function TimezoneSelect({ value, onChange }: { value: string; onChange: (zone: string) => void }) {
    return (
        <select
            value={value}
            onChange={(event) => onChange(event.target.value)}
            className="h-[38px] min-h-[44px] w-full rounded-lg border border-input bg-card px-3 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring sm:min-h-0"
        >
            {timeZones(value).map((zone) => (
                <option key={zone} value={zone}>
                    {zone.replace(/_/g, ' ')}
                </option>
            ))}
        </select>
    )
}

export default TimezoneSelect
