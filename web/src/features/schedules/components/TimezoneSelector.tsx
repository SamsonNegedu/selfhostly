import React, { useState, useMemo } from 'react';
import { Search, ChevronDown } from 'lucide-react';

const labelCache = new Map<string, string>();

const FALLBACK_TIMEZONES = [
  'Europe/London',
  'Europe/Paris',
  'Europe/Berlin',
  'Europe/Madrid',
  'Europe/Rome',
  'Europe/Amsterdam',
  'Europe/Vienna',
  'UTC',
];

/**
 * Formats a timezone string into a human-readable label
 * @param timezone - IANA timezone string (e.g., "America/New_York")
 * @returns Formatted label (e.g., "New York (EST)")
 */
const getTimezoneLabel = (timezone: string): string => {
  if (labelCache.has(timezone)) {
    return labelCache.get(timezone)!;
  }

  try {
    const now = new Date();
    const formatter = new Intl.DateTimeFormat('en-US', {
      timeZone: timezone,
      timeZoneName: 'short',
    });
    const parts = formatter.formatToParts(now);
    const timeZoneName = parts.find((part) => part.type === 'timeZoneName')?.value || '';
    const cityName = timezone.split('/').pop()?.replace(/_/g, ' ') || timezone;
    const label = `${cityName} (${timeZoneName})`;
    labelCache.set(timezone, label);
    return label;
  } catch {
    labelCache.set(timezone, timezone);
    return timezone;
  }
};

/**
 * Gets the current offset for a timezone
 * @param timezone IANA timezone string (e.g., "America/New_York")
 * @returns Formatted offset string (e.g., "-05:00")
 */
const getTimezoneOffset = (timezone: string): string => {
  try {
    const now = new Date();
    const utcDate = new Date(now.toLocaleString('en-US', { timeZone: 'UTC' }));
    const localDate = new Date(now.toLocaleString('en-US', { timeZone: timezone }));
    const offset = (localDate.getTime() - utcDate.getTime()) / (1000 * 60 * 60);
    const offsetHours = Math.floor(Math.abs(offset));
    const offsetMinutes = Math.floor((Math.abs(offset) % 1) * 60);

    const sign = offset >= 0 ? '+' : '-';
    const formattedHours = offsetHours.toString().padStart(2, '0');
    const formattedMinutes = offsetMinutes.toString().padStart(2, '0');

    return `${sign}${formattedHours}:${formattedMinutes}`;
  } catch {
    return '+00:00';
  }
};

/**
 * Gets all available timezones, sorted alphabetically by their display labels
 * Uses browser's Intl API when available, falls back to curated list
 * @returns Array of IANA timezone strings
 */
const getAvailableTimezones = (): string[] => {
  let timezones: string[];

  if (typeof Intl !== 'undefined' && 'supportedValuesOf' in Intl) {
    try {
      timezones = (Intl as unknown as { supportedValuesOf: (key: string) => string[] }).supportedValuesOf(
        'timeZone'
      );
    } catch {
      timezones = FALLBACK_TIMEZONES;
    }
  } else {
    timezones = FALLBACK_TIMEZONES;
  }

  return timezones.sort((a, b) => {
    const labelA = getTimezoneLabel(a);
    const labelB = getTimezoneLabel(b);
    return labelA.localeCompare(labelB);
  });
};

/**
 * Gets the user's current timezone from the browser
 * @returns IANA timezone string (e.g., "America/New_York")
 */
const getCurrentTimezone = (): string => {
  return Intl.DateTimeFormat().resolvedOptions().timeZone;
};

interface TimezoneSelectorProps {
  value: string;
  onChange: (timezone: string) => void;
  placeholder?: string;
  disabled?: boolean;
}

export function TimezoneSelector({ value, onChange, placeholder = 'Select timezone', disabled }: TimezoneSelectorProps) {
  const [isOpen, setIsOpen] = useState(false);
  const [searchTerm, setSearchTerm] = useState('');

  React.useEffect(() => {
    if (!value) {
      const browserTimezone = getCurrentTimezone();
      onChange(browserTimezone);
    }
  }, [value, onChange]);

  const availableTimezones = useMemo(() => getAvailableTimezones(), []);

  const groupedTimezones = useMemo(() => {
    const filtered = availableTimezones.filter(tz =>
      tz.toLowerCase().includes(searchTerm.toLowerCase()) ||
      getTimezoneLabel(tz).toLowerCase().includes(searchTerm.toLowerCase())
    );

    const groups: Record<string, string[]> = {};
    filtered.forEach(tz => {
      const parts = tz.split('/');
      const region = parts.length > 1 ? parts[0] : 'Other';
      if (!groups[region]) groups[region] = [];
      groups[region].push(tz);
    });

    return Object.entries(groups).map(([region, tzs]) => ({
      region,
      timezones: tzs.map((tz) => ({
        value: tz,
        label: getTimezoneLabel(tz),
        offset: getTimezoneOffset(tz),
      }))
    }));
  }, [availableTimezones, searchTerm]);

  const handleSelect = (timezone: string) => {
    onChange(timezone);
    setIsOpen(false);
    setSearchTerm('');
  };

  const selectedLabel = value ? getTimezoneLabel(value) : null;
  const selectedOffset = value ? getTimezoneOffset(value) : null;

  return (
    <div className="relative">
      <button
        type="button"
        onClick={() => setIsOpen(!isOpen)}
        disabled={disabled}
        className="w-full h-10 px-3 py-2 text-sm border border-input rounded-md bg-background text-foreground text-left flex items-center justify-between focus:outline-none focus:ring-2 focus:ring-ring disabled:opacity-50 disabled:cursor-not-allowed"
      >
        <span className="truncate">
          {selectedLabel ? (
            <span>{selectedLabel} ({selectedOffset})</span>
          ) : (
            <span className="text-muted-foreground">{placeholder}</span>
          )}
        </span>
        <ChevronDown className={`w-4 h-4 ml-2 flex-shrink-0 transition-transform ${isOpen ? 'rotate-180' : ''}`} />
      </button>

      {isOpen && (
        <div className="absolute z-10 bottom-full mb-1 w-full bg-background border border-border rounded-md shadow-lg max-h-60 overflow-hidden">
          {/* Search input */}
          <div className="p-2 border-b border-border">
            <div className="relative">
              <Search className="absolute left-2 top-1/2 transform -translate-y-1/2 w-4 h-4 text-muted-foreground" />
              <input
                type="text"
                placeholder="Search timezone..."
                className="w-full pl-8 pr-2 py-1 text-sm border border-input bg-background text-foreground rounded focus:outline-none focus:ring-2 focus:ring-ring"
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                autoFocus
              />
            </div>
          </div>

          {/* Timezones grouped by region */}
          <div className="max-h-48 overflow-y-auto">
            {groupedTimezones.map(({ region, timezones }) => (
              <div key={region}>
                <div className="px-3 py-1 text-xs font-medium text-muted-foreground bg-muted">
                  {region}
                </div>
                {timezones.map((tz) => (
                  <button
                    key={tz.value}
                    type="button"
                    onClick={() => handleSelect(tz.value)}
                    className={`w-full text-left px-3 py-2 text-sm hover:bg-accent hover:text-accent-foreground focus:bg-accent focus:outline-none ${
                      tz.value === value ? 'bg-accent text-accent-foreground font-medium' : ''
                    }`}
                  >
                    <div>
                      <div>{tz.label}</div>
                      <div className="text-xs text-muted-foreground">{tz.offset} • {tz.value}</div>
                    </div>
                  </button>
                ))}
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Close when clicking outside */}
      {isOpen && (
        <div
          className="fixed inset-0 z-0"
          onClick={() => setIsOpen(false)}
        />
      )}
    </div>
  );
}
