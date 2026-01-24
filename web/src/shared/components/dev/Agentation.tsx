import { useState, useEffect } from 'react';

/**
 * Agentation wrapper component for development environment only
 * Provides visual feedback tool for AI coding agents
 */
export function Agentation() {
    const [AgentationComponent, setAgentationComponent] = useState<any>(null);

    useEffect(() => {
        // Only load Agentation in development mode
        if (import.meta.env.DEV) {
            console.log('Loading Agentation in dev mode...');
            import('agentation')
                .then((module) => {
                    console.log('Agentation module loaded:', module);
                    // The agentation package exports Agentation as a named export component
                    if (module.Agentation) {
                        console.log('Setting Agentation component');
                        setAgentationComponent(() => module.Agentation);
                    }
                })
                .catch((error) => {
                    console.error('Failed to load Agentation:', error);
                });
        }
    }, []);

    // Render the Agentation component if it's loaded
    if (!AgentationComponent) {
        return null;
    }

    return <AgentationComponent />;
}
