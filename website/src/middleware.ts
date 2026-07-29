import { NextRequest, NextResponse } from 'next/server';

export function middleware(request: NextRequest) {
  if (request.nextUrl.pathname !== '/verify-email') {
    return NextResponse.next();
  }

  const token = request.nextUrl.searchParams.get('token');
  if (!token) {
    return NextResponse.next();
  }

  const redirectURL = request.nextUrl.clone();
  redirectURL.searchParams.delete('token');
  redirectURL.hash = `token=${encodeURIComponent(token)}`;

  return NextResponse.redirect(redirectURL, 307);
}

export const config = {
  matcher: ['/verify-email'],
};
