/** @type {import('next').NextConfig} */
const nextConfig = {
  output: 'standalone',
  reactStrictMode: true,
  poweredByHeader: false,
  // CSP + security headers are set in src/middleware.ts (nonce per request). Duplicating
  // script-src 'self' here breaks Next.js inline bootstrap scripts in production.
};

export default nextConfig;
