import { useState } from 'react'
import { Link } from 'react-router-dom'
import { ArrowLeft } from 'lucide-react'
import { buttonClasses } from '@/shared/components/ui/Button'
import { SegmentedControl } from '@/shared/components/ui/SegmentedControl'
import { ROUTES } from '@/shared/lib/routes'
import AutomaticJoin from './components/AutomaticJoin'
import ManualRegister from './components/ManualRegister'

type Mode = 'automatic' | 'manual'

const MODE_OPTIONS = [
    { value: 'automatic', label: 'Automatic' },
    { value: 'manual', label: 'Manual' },
]

function RegisterNodePage() {
    const [mode, setMode] = useState<Mode>('automatic')

    return (
        <div className="flex max-w-3xl flex-col gap-5">
            <div>
                <Link to={ROUTES.nodes} className={buttonClasses({ variant: 'ghost', className: '-ml-3' })}>
                    <ArrowLeft className="h-4 w-4" />
                    Nodes
                </Link>
                <h1 className="mt-1 text-2xl font-semibold tracking-tight">Add a node</h1>
                <p className="text-muted-foreground">Run apps on another machine and manage them from here.</p>
            </div>

            <SegmentedControl
                aria-label="How to add"
                options={MODE_OPTIONS}
                value={mode}
                onValueChange={(value) => setMode(value as Mode)}
                className="self-start"
            />

            {mode === 'automatic' ? <AutomaticJoin /> : <ManualRegister />}
        </div>
    )
}

export default RegisterNodePage
