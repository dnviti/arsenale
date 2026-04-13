import { render } from "@testing-library/react";
import { fireEvent, waitFor } from "@testing-library/dom";
import { MemoryRouter, Route, Routes, useLocation } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import LoginPage from "./LoginPage";
import { useAuthStore } from "../store/authStore";
import { useUiPreferencesStore } from "../store/uiPreferencesStore";

const {
  requestPasskeyOptionsApi,
  verifyPasskeyApi,
  loginApi,
  requestEmailCodeApi,
  verifyEmailCodeApi,
  verifyTotpApi,
  requestSmsCodeApi,
  verifySmsApi,
  mfaSetupInitApi,
  mfaSetupVerifyApi,
  requestWebAuthnOptionsApi,
  verifyWebAuthnApi,
} = vi.hoisted(() => ({
  requestPasskeyOptionsApi: vi.fn(),
  verifyPasskeyApi: vi.fn(),
  loginApi: vi.fn(),
  requestEmailCodeApi: vi.fn(),
  verifyEmailCodeApi: vi.fn(),
  verifyTotpApi: vi.fn(),
  requestSmsCodeApi: vi.fn(),
  verifySmsApi: vi.fn(),
  mfaSetupInitApi: vi.fn(),
  mfaSetupVerifyApi: vi.fn(),
  requestWebAuthnOptionsApi: vi.fn(),
  verifyWebAuthnApi: vi.fn(),
}));

const { getOAuthProviders } = vi.hoisted(() => ({
  getOAuthProviders: vi.fn(),
}));

const { resendVerificationEmail } = vi.hoisted(() => ({
  resendVerificationEmail: vi.fn(),
}));

const { switchTenant } = vi.hoisted(() => ({
  switchTenant: vi.fn(),
}));

const { browserSupportsWebAuthn, startAuthentication } = vi.hoisted(() => ({
  browserSupportsWebAuthn: vi.fn(),
  startAuthentication: vi.fn(),
}));

vi.mock("../api/auth.api", () => ({
  loginApi,
  requestPasskeyOptionsApi,
  verifyPasskeyApi,
  requestEmailCodeApi,
  verifyEmailCodeApi,
  verifyTotpApi,
  requestSmsCodeApi,
  verifySmsApi,
  mfaSetupInitApi,
  mfaSetupVerifyApi,
  requestWebAuthnOptionsApi,
  verifyWebAuthnApi,
}));

vi.mock("../api/oauth.api", () => ({
  getOAuthProviders,
}));

vi.mock("../api/email.api", () => ({
  resendVerificationEmail,
}));

vi.mock("../api/tenant.api", () => ({
  switchTenant,
}));

vi.mock("@simplewebauthn/browser", () => ({
  browserSupportsWebAuthn,
  startAuthentication,
}));

vi.mock("../components/OAuthButtons", () => ({
  default: () => <div data-testid="oauth-buttons" />,
}));

function HomeProbe() {
  const location = useLocation();
  return <div data-testid="home-probe">{location.pathname}{location.search}</div>;
}

function renderLoginPage(initialEntries: string[] = ["/login"]) {
  return render(
    <MemoryRouter initialEntries={initialEntries}>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route path="/" element={<HomeProbe />} />
      </Routes>
    </MemoryRouter>,
  );
}

describe("LoginPage", () => {
  beforeEach(() => {
    vi.resetAllMocks();
    localStorage.clear();

    useAuthStore.setState({
      accessToken: null,
      csrfToken: null,
      user: null,
      isAuthenticated: false,
      permissionsLoaded: false,
      permissionsLoading: false,
      permissionsSubject: null,
    });
    useUiPreferencesStore.setState({
      lastActiveTenantId: "",
    });

    getOAuthProviders.mockResolvedValue({ ldap: false });
    resendVerificationEmail.mockResolvedValue(undefined);
    switchTenant.mockResolvedValue({});
    browserSupportsWebAuthn.mockReturnValue(true);
    requestPasskeyOptionsApi.mockResolvedValue({
      tempToken: "temp-passkey-token",
      options: {
        challenge: "challenge-value",
        rpId: "localhost",
        timeout: 60000,
      },
    });
    startAuthentication.mockRejectedValue(
      new DOMException("cancelled", "NotAllowedError"),
    );
    loginApi.mockResolvedValue({
      accessToken: "access",
      csrfToken: "csrf",
      user: {
        id: "user-1",
        email: "admin@example.com",
        username: null,
        avatarData: null,
      },
      tenantMemberships: [],
    });
    requestEmailCodeApi.mockResolvedValue({ message: "sent" });
    verifyEmailCodeApi.mockResolvedValue({
      accessToken: "access",
      csrfToken: "csrf",
      user: {
        id: "user-1",
        email: "admin@example.com",
        username: null,
        avatarData: null,
      },
    });
    verifyTotpApi.mockResolvedValue({
      accessToken: "access",
      csrfToken: "csrf",
      user: {
        id: "user-1",
        email: "admin@example.com",
        username: null,
        avatarData: null,
      },
    });
    requestSmsCodeApi.mockResolvedValue({ message: "sent" });
    verifySmsApi.mockResolvedValue({
      accessToken: "access",
      csrfToken: "csrf",
      user: {
        id: "user-1",
        email: "admin@example.com",
        username: null,
        avatarData: null,
      },
    });
    mfaSetupInitApi.mockResolvedValue({
      secret: "secret",
      otpauthUri: "otpauth://totp/test",
    });
    mfaSetupVerifyApi.mockResolvedValue({
      accessToken: "access",
      csrfToken: "csrf",
      user: {
        id: "user-1",
        email: "admin@example.com",
        username: null,
        avatarData: null,
      },
    });
    requestWebAuthnOptionsApi.mockResolvedValue({ challenge: "challenge" });
    verifyWebAuthnApi.mockResolvedValue({
      accessToken: "access",
      csrfToken: "csrf",
      user: {
        id: "user-1",
        email: "admin@example.com",
        username: null,
        avatarData: null,
      },
    });
    verifyPasskeyApi.mockResolvedValue({
      accessToken: "access",
      csrfToken: "csrf",
      user: {
        id: "user-1",
        email: "admin@example.com",
        username: null,
        avatarData: null,
      },
      tenantMemberships: [],
    });
  });

  it("defaults to credentials when no previous login method is stored", async () => {
    const view = renderLoginPage();

    expect(
      await view.findByRole("button", { name: "Sign In" }),
    ).toBeInTheDocument();
    expect(view.getByRole("button", { name: "Try passkey instead" })).toBeInTheDocument();
  });

  it("starts in passkey-first mode and falls back after three failed attempts", async () => {
    useUiPreferencesStore.setState({ lastLoginMethod: "passkey" });
    const view = renderLoginPage();

    expect(
      await view.findByText(
        "Use a passkey to sign in without entering your email and password first.",
      ),
    ).toBeInTheDocument();

    await view.findByText("Failed attempts this visit: 1/3");

    fireEvent.click(view.getByRole("button", { name: "Retry Passkey" }));
    await view.findByText("Failed attempts this visit: 2/3");

    fireEvent.click(view.getByRole("button", { name: "Retry Passkey" }));

    expect(
      await view.findByRole("button", { name: "Try passkey instead" }),
    ).toBeInTheDocument();
    expect(view.getByRole("button", { name: "Sign In" })).toBeInTheDocument();
    expect(view.getByText("Forgot password?")).toBeInTheDocument();
    expect(requestPasskeyOptionsApi).toHaveBeenCalledTimes(3);
  });

  it("reveals password fallback immediately when the user chooses it", async () => {
    useUiPreferencesStore.setState({ lastLoginMethod: "passkey" });
    const view = renderLoginPage();

    await view.findByText(
      "Use a passkey to sign in without entering your email and password first.",
    );

    fireEvent.click(
      view.getByRole("button", { name: "Use email and password instead" }),
    );

    expect(
      await view.findByRole("button", { name: "Try passkey instead" }),
    ).toBeInTheDocument();
    expect(view.getByRole("button", { name: "Sign In" })).toBeInTheDocument();

    fireEvent.click(view.getByRole("button", { name: "Try passkey instead" }));

    expect(
      await view.findByText(
        "Use a passkey to sign in without entering your email and password first.",
      ),
    ).toBeInTheDocument();
    await waitFor(() => {
      expect(requestPasskeyOptionsApi).toHaveBeenCalledTimes(2);
    });
  });

  it("remembers passkey preference after successful passkey login", async () => {
    useUiPreferencesStore.setState({ lastLoginMethod: "passkey" });
    startAuthentication.mockResolvedValueOnce({ id: "cred-1" });
    const view = renderLoginPage();

    await waitFor(() => {
      expect(verifyPasskeyApi).toHaveBeenCalled();
    });

    expect(useUiPreferencesStore.getState().lastLoginMethod).toBe("passkey");
  });

  it("remembers credentials preference when user switches to password", async () => {
    useUiPreferencesStore.setState({ lastLoginMethod: "passkey" });
    const view = renderLoginPage();

    await view.findByText(
      "Use a passkey to sign in without entering your email and password first.",
    );

    fireEvent.click(
      view.getByRole("button", { name: "Use email and password instead" }),
    );

    expect(useUiPreferencesStore.getState().lastLoginMethod).toBe("credentials");
  });

  it("preserves supported deep-link actions after a successful sign-in", async () => {
    useUiPreferencesStore.setState({ lastLoginMethod: "passkey" });
    const view = renderLoginPage(["/login?action=open-settings"]);

    await view.findByText(
      "Use a passkey to sign in without entering your email and password first.",
    );

    fireEvent.click(
      view.getByRole("button", { name: "Use email and password instead" }),
    );

    fireEvent.change(await view.findByRole("textbox"), {
      target: { value: "admin@example.com" },
    });
    fireEvent.change(view.container.querySelector('input[type="password"]')!, {
      target: { value: "ArsenaleTemp91Qx" },
    });

    fireEvent.click(view.getByRole("button", { name: "Sign In" }));

    await waitFor(() => {
      expect(view.getByTestId("home-probe")).toHaveTextContent("/?action=open-settings");
    });
  });
});
