import { Request, Response, NextFunction } from 'express';
import { verify } from 'jsonwebtoken';

interface TokenPayload {
  userId: string;
  role: string;
  exp: number;
}

/** Validates JWT tokens and attaches user info to the request. */
export function authenticate(req: Request, res: Response, next: NextFunction) {
  const header = req.headers.authorization;
  if (!header) {
    res.status(401).json({ error: 'Missing token' });
    return;
  }

  const token = header.replace('Bearer ', '');
  try {
    const payload = verify(token, process.env.JWT_SECRET!) as TokenPayload;
    req.user = payload;
    next();
  } catch (e) {
    res.status(403).json({ error: 'Invalid token' });
  }
}

/** Checks if the authenticated user has the required role. */
export function requireRole(role: string) {
  return function checkRole(req: Request, res: Response, next: NextFunction) {
    if (req.user?.role !== role) {
      res.status(403).json({ error: 'Forbidden' });
      return;
    }
    next();
  };
}
