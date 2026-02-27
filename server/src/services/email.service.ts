import nodemailer from 'nodemailer';
import { config } from '../config';
import { logger } from '../utils/logger';

let transporter: nodemailer.Transporter | null = null;

function getTransporter(): nodemailer.Transporter | null {
  if (transporter) return transporter;
  if (!config.smtpHost) return null;

  transporter = nodemailer.createTransport({
    host: config.smtpHost,
    port: config.smtpPort,
    secure: config.smtpPort === 465,
    auth: {
      user: config.smtpUser,
      pass: config.smtpPass,
    },
  });
  return transporter;
}

export async function sendVerificationEmail(
  to: string,
  token: string,
): Promise<void> {
  const verifyUrl = `${config.clientUrl}/api/auth/verify-email?token=${token}`;

  const transport = getTransporter();
  if (!transport) {
    logger.info('========================================');
    logger.info('EMAIL VERIFICATION LINK (dev mode):');
    logger.info(verifyUrl);
    logger.info('========================================');
    return;
  }

  await transport.sendMail({
    from: config.smtpFrom,
    to,
    subject: 'Verify your email — Remote Desktop Manager',
    html: `
      <h2>Email Verification</h2>
      <p>Click the link below to verify your email address:</p>
      <p><a href="${verifyUrl}">${verifyUrl}</a></p>
      <p>This link expires in 24 hours.</p>
      <p>If you did not create an account, you can ignore this email.</p>
    `,
    text: `Verify your email: ${verifyUrl}\n\nThis link expires in 24 hours. If you did not create an account, ignore this email.`,
  });
}
