import { resolveDlpPolicy } from './dlp';

const allFalseTenant = {
  dlpDisableCopy: false,
  dlpDisablePaste: false,
  dlpDisableDownload: false,
  dlpDisableUpload: false,
};

const allTrueTenant = {
  dlpDisableCopy: true,
  dlpDisablePaste: true,
  dlpDisableDownload: true,
  dlpDisableUpload: true,
};

describe('resolveDlpPolicy', () => {
  it('returns all false when tenant is all false and no connection DLP', () => {
    const result = resolveDlpPolicy(allFalseTenant);
    expect(result).toEqual({
      disableCopy: false,
      disablePaste: false,
      disableDownload: false,
      disableUpload: false,
    });
  });

  it('disables copy when tenant disables copy regardless of connection', () => {
    const tenant = { ...allFalseTenant, dlpDisableCopy: true };
    const result = resolveDlpPolicy(tenant, { disableCopy: false });
    expect(result.disableCopy).toBe(true);
  });

  it('disables paste when connection disables paste regardless of tenant', () => {
    const result = resolveDlpPolicy(allFalseTenant, { disablePaste: true });
    expect(result.disablePaste).toBe(true);
  });

  it('disables download when both tenant and connection disable it', () => {
    const tenant = { ...allFalseTenant, dlpDisableDownload: true };
    const result = resolveDlpPolicy(tenant, { disableDownload: true });
    expect(result.disableDownload).toBe(true);
  });

  it('applies only tenant restrictions when connectionDlp is null', () => {
    const tenant = { ...allFalseTenant, dlpDisableCopy: true };
    const result = resolveDlpPolicy(tenant, null);
    expect(result).toEqual({
      disableCopy: true,
      disablePaste: false,
      disableDownload: false,
      disableUpload: false,
    });
  });

  it('applies only tenant restrictions when connectionDlp is undefined', () => {
    const tenant = { ...allFalseTenant, dlpDisablePaste: true };
    const result = resolveDlpPolicy(tenant, undefined);
    expect(result).toEqual({
      disableCopy: false,
      disablePaste: true,
      disableDownload: false,
      disableUpload: false,
    });
  });

  it('reflects mixed restrictions from tenant and connection', () => {
    const tenant = { ...allFalseTenant, dlpDisableCopy: true };
    const result = resolveDlpPolicy(tenant, { disableUpload: true });
    expect(result).toEqual({
      disableCopy: true,
      disablePaste: false,
      disableDownload: false,
      disableUpload: true,
    });
  });

  it('returns all true when both tenant and connection are all true', () => {
    const conn = {
      disableCopy: true,
      disablePaste: true,
      disableDownload: true,
      disableUpload: true,
    };
    const result = resolveDlpPolicy(allTrueTenant, conn);
    expect(result).toEqual({
      disableCopy: true,
      disablePaste: true,
      disableDownload: true,
      disableUpload: true,
    });
  });
});
