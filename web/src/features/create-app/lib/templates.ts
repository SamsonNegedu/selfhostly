export interface AppTemplate {
    id: string
    name: string
    description: string
    image: string
    containerPort: number
    defaultHostPort: number
    // Where the app keeps its data inside the container. It goes in a named volume so it survives updates.
    dataPath: string
    // Settings the image needs to start and answer on any address.
    environment?: Record<string, string>
}

export const TEMPLATES: AppTemplate[] = [
    {
        id: 'uptime-kuma',
        name: 'Uptime Kuma',
        description: 'Monitor your services and get alerts.',
        image: 'louislam/uptime-kuma:1',
        containerPort: 3001,
        defaultHostPort: 3001,
        dataPath: '/app/data',
    },
    {
        id: 'vaultwarden',
        name: 'Vaultwarden',
        description: 'A Bitwarden compatible password manager.',
        image: 'vaultwarden/server:latest',
        containerPort: 80,
        defaultHostPort: 8082,
        dataPath: '/data',
    },
    {
        id: 'jellyfin',
        name: 'Jellyfin',
        description: 'Stream your own movies, shows and music.',
        image: 'jellyfin/jellyfin:latest',
        containerPort: 8096,
        defaultHostPort: 8096,
        dataPath: '/config',
    },
    {
        id: 'gitea',
        name: 'Gitea',
        description: 'Git hosting with issues and pull requests.',
        image: 'gitea/gitea:latest',
        containerPort: 3000,
        defaultHostPort: 3000,
        dataPath: '/data',
    },
    {
        id: 'homepage',
        name: 'Homepage',
        description: 'A start page for all your services.',
        image: 'ghcr.io/gethomepage/homepage:latest',
        containerPort: 3000,
        defaultHostPort: 3010,
        dataPath: '/app/config',
        environment: { HOMEPAGE_ALLOWED_HOSTS: '*' },
    },
    {
        id: 'memos',
        name: 'Memos',
        description: 'Quick notes you keep on your own server.',
        image: 'neosmemo/memos:stable',
        containerPort: 5230,
        defaultHostPort: 5230,
        dataPath: '/var/opt/memos',
    },
    {
        id: 'linkding',
        name: 'Linkding',
        description: 'Save and search the links you care about.',
        image: 'sissbruecken/linkding:latest',
        containerPort: 9090,
        defaultHostPort: 9090,
        dataPath: '/etc/linkding/data',
    },
    {
        id: 'filebrowser',
        name: 'File Browser',
        description: 'Browse and share files from a web page.',
        image: 'filebrowser/filebrowser:latest',
        containerPort: 80,
        defaultHostPort: 8085,
        dataPath: '/srv',
    },
]

// The compose file for a template on a chosen host port. It is plain compose, so it can be edited afterwards.
export function templateCompose(template: AppTemplate, appName: string, hostPort: number): string {
    const volume = `${appName || template.id}_data`
    return [
        'services:',
        '  app:',
        `    image: ${template.image}`,
        '    restart: unless-stopped',
        ...(template.environment
            ? [
                  '    environment:',
                  ...Object.entries(template.environment).map(([key, value]) => `      ${key}: "${value}"`),
              ]
            : []),
        '    ports:',
        `      - "${hostPort}:${template.containerPort}"`,
        '    volumes:',
        `      - ${volume}:${template.dataPath}`,
        '',
        'volumes:',
        `  ${volume}:`,
        '',
    ].join('\n')
}

// GitHub links to a file page show HTML. The raw address returns the file itself.
export function toRawUrl(input: string): string {
    const trimmed = input.trim()
    const blob = trimmed.match(/^https:\/\/github\.com\/([^/]+)\/([^/]+)\/blob\/(.+)$/)
    return blob ? `https://raw.githubusercontent.com/${blob[1]}/${blob[2]}/${blob[3]}` : trimmed
}
