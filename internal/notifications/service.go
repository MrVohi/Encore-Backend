package notifications

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"
)

type ConcertJob struct {
	ConcertID  string
	ArtistID   string
	ArtistName string
	When       time.Time
	City       string
	Country    string
}

type Service struct {
	repo        *Repository
	sender      EmailSender
	frontendURL string
	queue       chan ConcertJob
}

func NewService(repo *Repository, sender EmailSender, frontendURL string) *Service {
	return &Service{
		repo:        repo,
		sender:      sender,
		frontendURL: strings.TrimRight(frontendURL, "/"),
		queue:       make(chan ConcertJob, 100),
	}
}

func (s *Service) Start(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case job := <-s.queue:
				s.handleConcert(ctx, job)
			}
		}
	}()
}

func (s *Service) EnqueueConcert(job ConcertJob) {
	select {
	case s.queue <- job:
	default:
		log.Printf("notifications queue full; dropping concert=%s", job.ConcertID)
	}
}

func (s *Service) handleConcert(ctx context.Context, job ConcertJob) {
	followers, err := s.repo.ListFollowersForArtist(ctx, job.ArtistID)
	if err != nil {
		log.Printf("notifications: list followers failed: %v", err)
		return
	}
	log.Printf("notifications: concert=%s artist=%s followers=%d", job.ConcertID, job.ArtistID, len(followers))

	for _, f := range followers {
		log.Printf("notifications: follower user=%s email=%s verified=%v", f.UserID, f.Email, f.EmailVerified)
		if !f.EmailVerified || f.Email == "" {
			continue
		}

		inserted, err := s.repo.TryMarkNotified(ctx, job.ConcertID, f.UserID)
		if err != nil {
			log.Printf("notifications: mark notified failed: %v", err)
			continue
		}
		if !inserted {
			continue
		}

		subject := fmt.Sprintf("New concert announced for %s", job.ArtistName)
		link := fmt.Sprintf("%s/concerts/%s", s.frontendURL, job.ConcertID)
		date := job.When.Format("Mon, 02 Jan 2006 at 15:04")

		preheader := fmt.Sprintf("%s just announced a new date.", job.ArtistName)
		location := fmt.Sprintf("%s, %s", job.City, job.Country)

		text := fmt.Sprintf(
			"Encore\n\nNew concert announced for %s\nDate: %s\nLocation: %s\n\nView details: %s\n\nYou are receiving this because you follow %s.\nIf this wasn't you, contact support.\n",
			job.ArtistName,
			date,
			location,
			link,
			job.ArtistName,
		)

		html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>New concert announced</title>
  </head>
  <body style="margin:0;padding:0;background:#f5f6f8;font-family:Arial,Helvetica,sans-serif;color:#111827;">
    <div style="display:none;max-height:0;overflow:hidden;color:transparent;opacity:0;">
      %s
    </div>
    <table role="presentation" width="100%%" cellpadding="0" cellspacing="0" style="background:#f5f6f8;padding:32px 16px;">
      <tr>
        <td align="center">
          <table role="presentation" width="600" cellpadding="0" cellspacing="0" style="width:100%%;max-width:600px;background:#ffffff;border-radius:12px;overflow:hidden;box-shadow:0 6px 18px rgba(17,24,39,0.08);">
            <tr>
              <td style="padding:24px 28px;background:#0f172a;color:#ffffff;">
                <div style="font-size:18px;font-weight:700;letter-spacing:0.4px;">Encore</div>
                <div style="font-size:12px;color:#e2e8f0;margin-top:4px;">Concert notifications</div>
              </td>
            </tr>
            <tr>
              <td style="padding:28px 28px 10px;">
                <h1 style="margin:0 0 12px;font-size:22px;color:#111827;">New concert announced</h1>
                <p style="margin:0 0 16px;font-size:15px;color:#374151;line-height:1.5;">
                  <strong>%s</strong> just announced a new concert.
                </p>
                <table role="presentation" width="100%%" cellpadding="0" cellspacing="0" style="margin:0 0 18px;">
                  <tr>
                    <td style="padding:12px 14px;background:#f8fafc;border:1px solid #e5e7eb;border-radius:10px;">
                      <div style="font-size:13px;color:#6b7280;text-transform:uppercase;letter-spacing:0.08em;">Date</div>
                      <div style="font-size:16px;font-weight:600;color:#111827;margin-top:4px;">%s</div>
                      <div style="font-size:13px;color:#6b7280;text-transform:uppercase;letter-spacing:0.08em;margin-top:12px;">Location</div>
                      <div style="font-size:16px;font-weight:600;color:#111827;margin-top:4px;">%s</div>
                    </td>
                  </tr>
                </table>
                <a href="%s" style="display:inline-block;background:#2563eb;color:#ffffff;text-decoration:none;padding:12px 20px;border-radius:8px;font-weight:600;font-size:14px;">
                  View concert details
                </a>
              </td>
            </tr>
            <tr>
              <td style="padding:22px 28px 28px;font-size:12px;color:#6b7280;line-height:1.6;">
                You are receiving this email because you follow <strong>%s</strong> on Encore.
                <br/>
                If this wasn't you, please contact support.
              </td>
            </tr>
          </table>
        </td>
      </tr>
    </table>
  </body>
</html>`,
			preheader,
			job.ArtistName,
			date,
			location,
			link,
			job.ArtistName,
		)

		if err := s.sender.Send(f.Email, subject, text, html); err != nil {
			log.Printf("notifications: send failed to %s: %v", f.Email, err)
		}
	}
}
