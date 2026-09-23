import { ChangeDetectionStrategy, ChangeDetectorRef, Component, DestroyRef, OnInit, inject } from "@angular/core";
import { takeUntilDestroyed } from "@angular/core/rxjs-interop";
import { ActivatedRoute, Router } from '@angular/router';
import { ChartResult, ConfigService, FeatureID, Netquery, PortapiService, Record as PortmasterRecord, SPNService, SPNStatus, StringSetting, UserProfile } from "@safing/portmaster-api";
import { SfngDialogService } from '@safing/ui';
import { catchError, concatMap, finalize, forkJoin, from, interval, map, of, startWith, switchMap } from "rxjs";
import { fadeInAnimation, fadeOutAnimation } from "../animations";
import { SPNAccountDetailsComponent } from '../spn-account-details';

interface WireGuardStatus extends PortmasterRecord {
  State: 'disabled' | 'connecting' | 'connected' | 'failed';
  LastError: string;
  ConnectedSince: string | null;
}

@Component({
  selector: 'app-spn-status',
  templateUrl: './spn-status.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
  animations: [
    fadeInAnimation,
    fadeOutAnimation,
  ]
})
export class SPNStatusComponent implements OnInit {
  private destroyRef = inject(DestroyRef);

  tunnelMode: 'off' | 'spn' | 'wireguard' = 'off';

  wireGuardStatus: WireGuardStatus | null = null;
  wireGuardImporting = false;
  wireGuardImportFailed = false;
  wireGuardImportMessage = '';

  get spnEnabled() { return this.tunnelMode === 'spn'; }

  /** The chart data for the SPN connection chart */
  spnConnChart: ChartResult[] = [];

  /** The current amount of SPN identities used */
  identities: number = 0;

  /** The current SPN user profile */
  profile: UserProfile | null = null;

  /** The current status of the SPN module */
  spnStatus: SPNStatus | null = null;

  /** Returns whether or not the current package has the SPN feature */
  get packageHasSPN() {
    return this.profile?.current_plan?.feature_ids?.includes(FeatureID.SPN)
  }

  constructor(
    private configService: ConfigService,
    private portapi: PortapiService,
    private spnService: SPNService,
    private netquery: Netquery,
    private cdr: ChangeDetectorRef,
    private router: Router,
    private activeRoute: ActivatedRoute,
    private dialog: SfngDialogService
  ) { }

  ngOnInit(): void {
    this.spnService
      .profile$
      .pipe(
        takeUntilDestroyed(this.destroyRef),
        catchError(() => of(null))
      )
      .subscribe(profile => {
        this.profile = profile || null;

        this.cdr.markForCheck();
      });

    this.spnService.status$
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe(status => {
        this.spnStatus = status;

        this.cdr.markForCheck();
      })

    this.configService.watch<StringSetting>("network/tunnel/mode")
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe(value => {
        this.tunnelMode = value as typeof this.tunnelMode;

        // If the user disabled the SPN clear the connection chart
        // as well.
        if (!this.spnEnabled) {
          this.spnConnChart = [];
        }

        this.cdr.markForCheck();
      });

    this.portapi.watch<WireGuardStatus>('runtime:wireguard/status', { ignoreDelete: true })
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe(status => {
        this.wireGuardStatus = status;
        this.cdr.markForCheck();
      });

    interval(5000)
      .pipe(
        startWith(-1),
        takeUntilDestroyed(this.destroyRef),
        switchMap(() => forkJoin({
          chart: this.netquery.activeConnectionChart({ tunneled: { $eq: true } }),
          identities: this.netquery.query({
            query: { tunneled: { $eq: true }, exit_node: { $ne: "" } },
            groupBy: ['exit_node'],
            select: [
              'exit_node',
              { $count: { field: '*', as: 'totalCount' } }
            ]
          }, 'spn-status-get-connections-count-per-exit-node')
        }))
      )
      .subscribe(data => {
        this.spnConnChart = data.chart;
        this.identities = data.identities.length;

        this.cdr.markForCheck();
      })
  }

  openOrLogin() {
    if (this.activeRoute.snapshot.firstChild?.url[0]?.path === "spn") {
      this.dialog.create(SPNAccountDetailsComponent, {
        autoclose: true,
        backdrop: 'light'
      })

      return
    }

    this.router.navigate(['/spn'])
  }

  setTunnelMode(mode: 'off' | 'spn' | 'wireguard') {
    if (mode === 'spn' && !this.packageHasSPN) {
      this.openOrLogin();
      return;
    }
    this.configService.save('network/tunnel/mode', mode)
      .subscribe();
  }

  importWireGuardProfile(event: Event) {
    const input = event.target as HTMLInputElement;
    const file = input.files?.item(0);
    input.value = '';
    if (!file) {
      return;
    }

    // WireGuard profiles are only a few KiB. The limit avoids retaining an
    // accidentally selected large file as a sensitive configuration value.
    if (file.size === 0 || file.size > 256 * 1024) {
      this.setWireGuardImportResult(false, 'Select a non-empty WireGuard profile smaller than 256 KiB.');
      return;
    }

    this.wireGuardImporting = true;
    this.wireGuardImportFailed = false;
    this.wireGuardImportMessage = 'Importing WireGuard profile ...';
    this.cdr.markForCheck();

    from(file.text())
      .pipe(
        map(profile => {
          if (!profile.trim()) {
            throw new Error('empty profile');
          }
          return profile;
        }),
        concatMap(profile => this.configService.save('wireguard/config', profile)),
        concatMap(() => this.configService.save('network/tunnel/mode', 'wireguard')),
        takeUntilDestroyed(this.destroyRef),
        finalize(() => {
          this.wireGuardImporting = false;
          this.cdr.markForCheck();
        }),
      )
      .subscribe({
        complete: () => this.setWireGuardImportResult(true, 'WireGuard profile imported. Connecting ...'),
        error: () => this.setWireGuardImportResult(false, 'The WireGuard profile could not be imported.'),
      });
  }

  private setWireGuardImportResult(success: boolean, message: string) {
    this.wireGuardImportFailed = !success;
    this.wireGuardImportMessage = message;
    this.cdr.markForCheck();
  }
}
