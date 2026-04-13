import type { ReactNode } from 'react';
import { AlertTriangle } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import type { AuditGateway } from '../../api/audit.api';
import type { TenantUser } from '../../api/tenant.api';
import { ACTION_LABELS, ALL_ACTIONS, TARGET_TYPES } from '../Audit/auditConstants';
import { ALL_VALUE } from '../Settings/tenantAuditLogUtils';

function FilterField({
  children,
  label,
}: {
  children: ReactNode;
  label: string;
}) {
  return (
    <div className="space-y-2">
      <Label>{label}</Label>
      {children}
    </div>
  );
}

interface AuditLogTenantFiltersProps {
  action: string;
  countries: string[];
  endDate: string;
  flaggedOnly: boolean;
  gatewayId: string;
  gateways: AuditGateway[];
  geoCountry: string;
  ipAddress: string;
  onActionChange: (value: string) => void;
  onClearFilters: () => void;
  onCountryChange: (value: string) => void;
  onEndDateChange: (value: string) => void;
  onFlaggedToggle: () => void;
  onGatewayChange: (value: string) => void;
  onIpAddressChange: (value: string) => void;
  onSortByChange: (value: string) => void;
  onSortOrderChange: (value: string) => void;
  onStartDateChange: (value: string) => void;
  onTargetTypeChange: (value: string) => void;
  onUserChange: (value: string) => void;
  sortBy: string;
  sortOrder: string;
  startDate: string;
  targetType: string;
  userId: string;
  users: TenantUser[];
}

export default function AuditLogTenantFilters({
  action,
  countries,
  endDate,
  flaggedOnly,
  gatewayId,
  gateways,
  geoCountry,
  ipAddress,
  onActionChange,
  onClearFilters,
  onCountryChange,
  onEndDateChange,
  onFlaggedToggle,
  onGatewayChange,
  onIpAddressChange,
  onSortByChange,
  onSortOrderChange,
  onStartDateChange,
  onTargetTypeChange,
  onUserChange,
  sortBy,
  sortOrder,
  startDate,
  targetType,
  userId,
  users,
}: AuditLogTenantFiltersProps) {
  return (
    <div className="space-y-4 border-t pt-4">
      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        <FilterField label="User">
          <Select value={userId || ALL_VALUE} onValueChange={(value) => onUserChange(value === ALL_VALUE ? '' : value)}>
            <SelectTrigger>
              <SelectValue placeholder="All users" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value={ALL_VALUE}>All users</SelectItem>
              {users.map((user) => (
                <SelectItem key={user.id} value={user.id}>
                  {user.username ?? user.email}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </FilterField>

        <FilterField label="Action">
          <Select value={action || ALL_VALUE} onValueChange={(value) => onActionChange(value === ALL_VALUE ? '' : value)}>
            <SelectTrigger>
              <SelectValue placeholder="All actions" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value={ALL_VALUE}>All actions</SelectItem>
              {ALL_ACTIONS.map((entry) => (
                <SelectItem key={entry} value={entry}>
                  {ACTION_LABELS[entry]}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </FilterField>

        <FilterField label="Target type">
          <Select value={targetType || ALL_VALUE} onValueChange={(value) => onTargetTypeChange(value === ALL_VALUE ? '' : value)}>
            <SelectTrigger>
              <SelectValue placeholder="All target types" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value={ALL_VALUE}>All target types</SelectItem>
              {TARGET_TYPES.map((type) => (
                <SelectItem key={type} value={type}>
                  {type}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </FilterField>

        <FilterField label="Gateway">
          <Select value={gatewayId || ALL_VALUE} onValueChange={(value) => onGatewayChange(value === ALL_VALUE ? '' : value)}>
            <SelectTrigger>
              <SelectValue placeholder="All gateways" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value={ALL_VALUE}>All gateways</SelectItem>
              {gateways.map((gateway) => (
                <SelectItem key={gateway.id} value={gateway.id}>
                  {gateway.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </FilterField>

        <FilterField label="IP address">
          <Input
            value={ipAddress}
            placeholder="8.8.8.8"
            onChange={(event) => onIpAddressChange(event.target.value)}
          />
        </FilterField>

        <FilterField label="Country">
          <Select value={geoCountry || ALL_VALUE} onValueChange={(value) => onCountryChange(value === ALL_VALUE ? '' : value)}>
            <SelectTrigger>
              <SelectValue placeholder="All countries" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value={ALL_VALUE}>All countries</SelectItem>
              {countries.map((country) => (
                <SelectItem key={country} value={country}>
                  {country}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </FilterField>

        <FilterField label="From">
          <Input type="date" value={startDate} onChange={(event) => onStartDateChange(event.target.value)} />
        </FilterField>

        <FilterField label="To">
          <Input type="date" value={endDate} onChange={(event) => onEndDateChange(event.target.value)} />
        </FilterField>

        <FilterField label="Sort by">
          <Select value={sortBy} onValueChange={onSortByChange}>
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="createdAt">Date</SelectItem>
              <SelectItem value="action">Action</SelectItem>
            </SelectContent>
          </Select>
        </FilterField>

        <FilterField label="Order">
          <Select value={sortOrder} onValueChange={onSortOrderChange}>
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="desc">Newest first</SelectItem>
              <SelectItem value="asc">Oldest first</SelectItem>
            </SelectContent>
          </Select>
        </FilterField>
      </div>

      <div className="flex flex-wrap gap-2">
        <Button type="button" size="sm" variant={flaggedOnly ? 'default' : 'outline'} onClick={onFlaggedToggle}>
          <AlertTriangle className="size-4" />
          Flagged Only
        </Button>
        <Button type="button" size="sm" variant="ghost" onClick={onClearFilters}>
          Clear Filters
        </Button>
      </div>
    </div>
  );
}
