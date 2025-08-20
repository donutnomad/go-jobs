import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  async rewrites() {
    // 从环境变量读取 API URL，默认为 localhost:8080
    const apiUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';
    return [
      {
        source: '/api/:path*',
        destination: `${apiUrl}/api/:path*`,
      },
    ];
  },
};

export default nextConfig;
