import { Link as RouterLink, type LinkProps } from 'react-router-dom';
import { cn } from '@/lib/utils';

export default function AuthLink({ className, ...props }: LinkProps) {
  return (
    <RouterLink
      className={cn('font-medium text-primary transition-colors hover:text-primary/80', className)}
      {...props}
    />
  );
}
