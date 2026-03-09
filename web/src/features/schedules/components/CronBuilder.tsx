import { useState, useEffect } from 'react';
import { Input } from '@/shared/components/ui/Input';
import { Cron } from 'croner';

// Predefined cron templates
const CRON_TEMPLATES = [
  {
    name: 'Every day at 9 AM',
    expression: '0 9 * * *',
    description: 'Daily at 09:00'
  },
  {
    name: 'Every weekday at 9 AM',
    expression: '0 9 * * 1-5',
    description: 'Mon-Fri at 09:00'
  },
  {
    name: 'Every day at 6 PM',
    expression: '0 18 * * *',
    description: 'Daily at 18:00'
  },
  {
    name: 'Every weekday at 6 PM',
    expression: '0 18 * * 1-5',
    description: 'Mon-Fri at 18:00'
  },
  {
    name: 'Every weekend at 10 AM',
    expression: '0 10 * * 0,6',
    description: 'Sat-Sun at 10:00'
  },
  {
    name: 'End of month (28th at midnight)',
    expression: '0 0 28 * *',
    description: 'Monthly on 28th'
  },
  {
    name: 'Start of month (1st at midnight)',
    expression: '0 0 1 * *',
    description: 'Monthly on 1st'
  },
  {
    name: '3rd day of month',
    expression: '0 0 3 * *',
    description: 'Monthly on 3rd'
  },
  {
    name: 'Every hour',
    expression: '0 * * * *',
    description: 'Start of every hour'
  },
];

// Cron format guide
const CRON_FORMAT = 'minute hour day month weekday';

interface CronBuilderProps {
  value: string;
  onChange: (expression: string) => void;
  placeholder?: string;
  disabled?: boolean;
  error?: string;
}

export function CronBuilder({ value, onChange, placeholder, disabled, error }: CronBuilderProps) {
  const [expression, setExpression] = useState(value || '');

  useEffect(() => {
    setExpression(value || '');
  }, [value]);

  const handleChange = (newValue: string) => {
    setExpression(newValue);
    onChange(newValue);
  };

  const handleTemplateSelect = (e: React.ChangeEvent<HTMLSelectElement>) => {
    const templateExpression = e.target.value;
    if (templateExpression) {
      handleChange(templateExpression);
    }
  };

  const parseExpression = (expr: string) => {
    if (!expr) return null;
    try {
      // Validate the cron expression using croner
      new Cron(expr, { paused: true });
      const parts = expr.trim().split(/\s+/);
      if (parts.length !== 5) return null;
      return {
        minute: parts[0],
        hour: parts[1],
        day: parts[2],
        month: parts[3],
        weekday: parts[4],
      };
    } catch {
      return null;
    }
  };

  const getHumanReadable = (expr: string): string | null => {
    if (!expr) return null;

    try {
      // Use croner to validate and get next run
      const cron = new Cron(expr, { paused: true });
      const nextRun = cron.nextRun();
      if (!nextRun) return null;

      const parts = expr.trim().split(/\s+/);
      if (parts.length !== 5) return null;

      const [minute, hour, day, , weekday] = parts;

      // Check for common patterns first
      const patterns: Record<string, string> = {
        '* * * * *': 'Every minute',
        '0 * * * *': 'Every hour',
        '0 0 * * *': 'Every day at midnight',
        '0 9 * * *': 'Every day at 9:00 AM',
        '0 18 * * *': 'Every day at 6:00 PM',
        '0 9 * * 1-5': 'Every weekday (Mon-Fri) at 9:00 AM',
        '0 18 * * 1-5': 'Every weekday (Mon-Fri) at 6:00 PM',
        '0 10 * * 0,6': 'Every weekend (Sat-Sun) at 10:00 AM',
        '0 9 1 * *': 'First day of every month at 9:00 AM',
        '0 0 1 * *': 'First day of every month at midnight',
        '0 0 28 * *': 'On the 28th of every month at midnight',
        '0 0 3 * *': 'On the 3rd of every month at midnight',
      };

      if (patterns[expr]) return patterns[expr];

      // Build dynamic interpretation
      let description = '';

      // Time of day
      if (minute !== '*' || hour !== '*') {
        if (minute.startsWith('*/')) {
          description += `Every ${minute.slice(2)} minutes`;
        } else if (hour.startsWith('*/')) {
          description += `Every ${hour.slice(2)} hours`;
        } else if (hour !== '*') {
          const hourNum = parseInt(hour);
          const period = hourNum >= 12 ? 'PM' : 'AM';
          const displayHour = hourNum > 12 ? hourNum - 12 : hourNum === 0 ? 12 : hourNum;
          description += `At ${displayHour}:${minute.padStart(2, '0')} ${period}`;
        } else {
          description += 'Every hour';
        }
      }

      // Day of week
      if (weekday !== '*') {
        const days: Record<string, string> = {
          '0': 'Sunday', '1': 'Monday', '2': 'Tuesday', '3': 'Wednesday',
          '4': 'Thursday', '5': 'Friday', '6': 'Saturday', '7': 'Sunday'
        };

        if (weekday === '1-5') {
          description += ' on weekdays';
        } else if (weekday === '0,6' || weekday === '6,0') {
          description += ' on weekends';
        } else if (weekday.includes(',')) {
          const dayNames = weekday.split(',').map(d => days[d] || d).join(', ');
          description += ` on ${dayNames}`;
        } else if (weekday.includes('-')) {
          const [start, end] = weekday.split('-');
          description += ` on ${days[start]}-${days[end]}`;
        } else {
          description += ` on ${days[weekday]}`;
        }
      } else if (day !== '*') {
        if (day === '1') {
          description += ' on the 1st of every month';
        } else if (day === '2') {
          description += ' on the 2nd of every month';
        } else if (day === '3') {
          description += ' on the 3rd of every month';
        } else if (day === '28') {
          description += ' on the 28th of every month';
        } else if (day === '29') {
          description += ' on the 29th of every month';
        } else if (day === '30') {
          description += ' on the 30th of every month';
        } else if (day === '31') {
          description += ' on the 31st of every month';
        } else if (day.includes(',')) {
          description += ` on days ${day} of every month`;
        } else if (day.includes('-')) {
          description += ` on days ${day} of every month`;
        } else {
          description += ` on day ${day} of every month`;
        }
      } else {
        description += ' every day';
      }

      return description || 'Custom schedule';
    } catch {
      return null;
    }
  };

  const parsed = parseExpression(expression);
  const humanReadable = getHumanReadable(expression);

  return (
    <div className="space-y-3">
      {/* Quick presets dropdown */}
      <select
        onChange={handleTemplateSelect}
        disabled={disabled}
        value=""
        className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50"
      >
        <option value="">Select preset...</option>
        {CRON_TEMPLATES.map((template) => (
          <option key={template.expression} value={template.expression}>
            {template.name} - {template.description}
          </option>
        ))}
      </select>

      {/* Input Field */}
      <Input
        type="text"
        value={expression}
        onChange={(e) => handleChange(e.target.value)}
        placeholder={placeholder || "0 9 * * 1-5"}
        disabled={disabled}
        className={error ? 'border-destructive' : ''}
      />

      {/* Format hint and parsed breakdown */}
      {parsed && (
        <div className="flex items-center justify-between text-xs text-muted-foreground">
          <span className="font-mono">{CRON_FORMAT}</span>
          <div className="flex items-center gap-1">
            <code className="font-mono bg-muted px-1.5 py-0.5 rounded">{parsed.minute}</code>
            <code className="font-mono bg-muted px-1.5 py-0.5 rounded">{parsed.hour}</code>
            <code className="font-mono bg-muted px-1.5 py-0.5 rounded">{parsed.day}</code>
            <code className="font-mono bg-muted px-1.5 py-0.5 rounded">{parsed.month}</code>
            <code className="font-mono bg-muted px-1.5 py-0.5 rounded">{parsed.weekday}</code>
          </div>
        </div>
      )}

      {/* Human readable interpretation */}
      {humanReadable && (
        <div className="flex items-center gap-2 px-3 py-2 bg-muted rounded-md text-sm">
          <span className="text-muted-foreground">Schedule:</span>
          <span className="font-medium">{humanReadable}</span>
        </div>
      )}

      {!parsed && expression && (
        <div className="text-xs text-muted-foreground">
          <span className="font-mono">{CRON_FORMAT}</span>
        </div>
      )}
    </div>
  );
}
