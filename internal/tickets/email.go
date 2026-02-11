package tickets

import (
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	qrcode "github.com/skip2/go-qrcode"
)

type TicketEmailInfo struct {
	ConcertTitle string
	When         time.Time
	City         string
	Country      string
}

func buildTicketEmail(info TicketEmailInfo, quantity int, qrPayload string) (subject, textBody, htmlBody string, err error) {
	date := info.When.Format("Mon, 02 Jan 2006 at 15:04")
	location := strings.TrimSpace(info.City)
	if info.Country != "" {
		if location != "" {
			location = location + ", " + info.Country
		} else {
			location = info.Country
		}
	}

	qrPNG, err := qrcode.Encode(qrPayload, qrcode.Medium, 256)
	if err != nil {
		return "", "", "", err
	}

	qrBase64 := base64.StdEncoding.EncodeToString(qrPNG)
	qrDataURL := "data:image/png;base64," + qrBase64

	subject = fmt.Sprintf("Your ticket for %s", info.ConcertTitle)

	textBody = fmt.Sprintf(
		"Encore\n\nTicket confirmation\nEvent: %s\nDate: %s\nLocation: %s\nQuantity: %d\n\nTicket code: %s\n\nShow this code at the venue.",
		info.ConcertTitle,
		date,
		location,
		quantity,
		qrPayload,
	)

	// Palette from frontend styles.css
	primary := "#2d2d2d"
	accentWarm := "#e07a5f"
	accentCool := "#3d5a80"
	accentLight := "#f4a261"
	bgCream := "#fdfcf9"
	bgLight := "#f8f5f1"
	textDark := "#1a1a1a"
	textMuted := "#6c757d"

	htmlBody = fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>Your ticket</title>
  </head>
  <body style="margin:0;padding:0;background:%s;font-family:'Segoe UI', Arial, Helvetica, sans-serif;color:%s;">
    <table role="presentation" width="100%%" cellpadding="0" cellspacing="0" style="background:%s;padding:32px 16px;">
      <tr>
        <td align="center">
          <table role="presentation" width="640" cellpadding="0" cellspacing="0" style="width:100%%;max-width:640px;background:%s;border:3px solid %s;border-radius:16px;overflow:hidden;box-shadow:8px 8px 0 %s;">
            <tr>
              <td style="padding:24px 28px;background:%s;color:#ffffff;">
                <div style="font-size:20px;font-weight:800;letter-spacing:0.5px;">Encore</div>
                <div style="font-size:12px;color:rgba(255,255,255,0.75);margin-top:4px;letter-spacing:0.08em;text-transform:uppercase;">Ticket confirmation</div>
              </td>
            </tr>
            <tr>
              <td style="padding:28px 28px 10px;">
                <h1 style="margin:0 0 10px;font-size:22px;color:%s;">Your ticket is confirmed</h1>
                <p style="margin:0 0 18px;font-size:15px;color:%s;line-height:1.5;">
                  You purchased <strong>%d</strong> ticket(s) for <strong>%s</strong>.
                </p>
                <table role="presentation" width="100%%" cellpadding="0" cellspacing="0" style="margin:0 0 18px;">
                  <tr>
                    <td style="padding:14px 16px;background:%s;border:2px solid %s;border-radius:12px;">
                      <div style="font-size:12px;color:%s;text-transform:uppercase;letter-spacing:0.08em;">Date</div>
                      <div style="font-size:16px;font-weight:700;color:%s;margin-top:4px;">%s</div>
                      <div style="font-size:12px;color:%s;text-transform:uppercase;letter-spacing:0.08em;margin-top:12px;">Location</div>
                      <div style="font-size:16px;font-weight:700;color:%s;margin-top:4px;">%s</div>
                    </td>
                  </tr>
                </table>
                <div style="text-align:center;padding:12px 0 8px;">
                  <div style="display:inline-block;padding:10px;background:%s;border:2px dashed %s;border-radius:14px;">
                    <img alt="Ticket QR" src="%s" style="width:200px;height:200px;border:1px solid %s;border-radius:10px;background:#ffffff;" />
                  </div>
                  <div style="font-size:12px;color:%s;margin-top:10px;">Ticket code: %s</div>
                </div>
                <div style="margin-top:16px;padding:10px 12px;border-left:4px solid %s;background:color-mix(in srgb, %s 15%%, #ffffff);color:%s;font-size:12px;">
                  Present this QR code at the venue to validate your ticket.
                </div>
              </td>
            </tr>
            <tr>
              <td style="padding:22px 28px 28px;font-size:12px;color:%s;line-height:1.6;">
                If this wasn’t you, please contact support.
              </td>
            </tr>
          </table>
        </td>
      </tr>
    </table>
  </body>
</html>`,
		bgCream,
		textDark,
		bgCream,
		bgCream,
		primary,
		primary,
		accentCool,
		textDark,
		textMuted,
		quantity,
		info.ConcertTitle,
		bgLight,
		primary,
		textMuted,
		textDark,
		date,
		textMuted,
		textDark,
		location,
		bgLight,
		accentWarm,
		qrDataURL,
		primary,
		textMuted,
		qrPayload,
		accentLight,
		accentLight,
		textDark,
		textMuted,
	)

	return subject, textBody, htmlBody, nil
}
