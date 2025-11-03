/** @type {import('next').NextConfig} */
const nextConfig = {
    reactStrictMode: true,
    swcMinify: false, // Disable SWC minification to avoid download issues
    output: 'standalone',
    experimental: {
        swcPlugins: [], // Disable SWC plugins
    },
    async rewrites() {
        return [
            {
                source: '/api/:path*',
                destination: 'http://localhost:8080/api/:path*',
            },
        ]
    },
    images: {
        domains: ['localhost'],
    },
}

module.exports = nextConfig