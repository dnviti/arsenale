import type { ReactNode } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { cn } from '@/lib/utils';

interface AuthLayoutProps {
  cardClassName?: string;
  children: ReactNode;
  contentClassName?: string;
  description?: ReactNode;
  descriptionClassName?: string;
  title: ReactNode;
  titleClassName?: string;
}

export default function AuthLayout({
  cardClassName,
  children,
  contentClassName,
  description,
  descriptionClassName,
  title,
  titleClassName,
}: AuthLayoutProps) {
  return (
    <div
      className="flex min-h-screen items-center justify-center bg-background px-4 py-10"
      style={{
        backgroundImage:
          'radial-gradient(ellipse at 50% 10%, color-mix(in srgb, var(--primary) 12%, transparent) 0%, var(--background) 70%)',
      }}
    >
      <Card className={cn(
        'w-full border-border/90 bg-card/95 shadow-[0_24px_80px_rgba(0,0,0,0.28)] backdrop-blur',
        cardClassName,
      )}
      >
        <CardHeader className="space-y-4 pb-4">
          <div className="flex justify-center">
            <div className="h-1 w-10 rounded-full bg-primary" />
          </div>
          <div className="space-y-2 text-center">
            <CardTitle className={cn('text-3xl sm:text-4xl', titleClassName)}>
              {title}
            </CardTitle>
            {description ? (
              <CardDescription className={cn('text-sm leading-6', descriptionClassName)}>
                {description}
              </CardDescription>
            ) : null}
          </div>
        </CardHeader>
        <CardContent className={cn('space-y-4', contentClassName)}>
          {children}
        </CardContent>
      </Card>
    </div>
  );
}
