import { DragDropModule } from '@angular/cdk/drag-drop';
import { OverlayModule } from '@angular/cdk/overlay';
import { PortalModule } from '@angular/cdk/portal';
import { ScrollingModule } from '@angular/cdk/scrolling';
import { CdkTableModule } from '@angular/cdk/table';
import { CommonModule } from '@angular/common';

import { APP_INITIALIZER, LOCALE_ID, NgModule } from '@angular/core';
import { FormsModule, ReactiveFormsModule } from '@angular/forms';
import { BrowserModule } from '@angular/platform-browser';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';
import { FaIconLibrary, FontAwesomeModule } from '@fortawesome/angular-fontawesome';
import { faGithub } from '@fortawesome/free-brands-svg-icons';
import { far } from '@fortawesome/free-regular-svg-icons';
import { fas } from '@fortawesome/free-solid-svg-icons';
import { PortmasterAPIModule } from '@safing/portmaster-api';
import { OverlayStepperModule, SfngAccordionModule, SfngDialogModule, SfngDropDownModule, SfngPaginationModule, SfngSelectModule, SfngTipUpModule, SfngToggleSwitchModule, SfngTooltipModule, TabModule, UiModule } from '@safing/ui';
import MyYamlFile from 'js-yaml-loader!../i18n/helptexts.yaml';
import * as i18n from 'ng-zorro-antd/i18n';
import { MarkdownModule } from 'ngx-markdown';
import { environment } from 'src/environments/environment';
import { AppRoutingModule } from './app-routing.module';
import { AppComponent } from './app.component';
import { IntroModule } from './intro';
import { NavigationComponent } from './layout/navigation/navigation';
import { SideDashComponent } from './layout/side-dash/side-dash';
import { AppOverviewComponent, AppViewComponent, QuickSettingInternetButtonComponent } from './pages/app-view';
import { QsHistoryComponent } from './pages/app-view/qs-history/qs-history.component';
import { QuickSettingSelectExitButtonComponent } from './pages/app-view/qs-select-exit/qs-select-exit';
import { QuickSettingUseSPNButtonComponent } from './pages/app-view/qs-use-spn/qs-use-spn';
import { QuickSettingUseSplitTunButtonComponent } from './pages/app-view/qs-use-splittun/qs-use-splittun';
import { DashboardPageComponent } from './pages/dashboard/dashboard.component';
import { FeatureCardComponent } from './pages/dashboard/feature-card/feature-card.component';
import { MonitorPageComponent } from './pages/monitor';
import { SettingsComponent } from './pages/settings/settings';
import { SPNModule } from './pages/spn/spn.module';
import { SupportPageComponent } from './pages/support';
import { SupportFormComponent } from './pages/support/form';
import { NotificationsService } from './services';
import { ActionIndicatorModule } from './shared/action-indicator';
import { SfngAppIconModule } from './shared/app-icon';
import { ConfigModule } from './shared/config';
import { CountIndicatorModule } from './shared/count-indicator';
import { CountryFlagModule } from './shared/country-flag';
import { EditProfileDialog } from './shared/edit-profile-dialog';
import { ExitScreenComponent } from './shared/exit-screen/exit-screen';
import { ExpertiseModule } from './shared/expertise/expertise.module';
import { ExternalLinkDirective } from './shared/external-link.directive';
import { FeatureScoutComponent } from './shared/feature-scout';
import { SfngFocusModule } from './shared/focus';
import { FuzzySearchPipe } from './shared/fuzzySearch';
import { LoadingComponent } from './shared/loading';
import { SfngMenuModule } from './shared/menu';
import { SfngMultiSwitchModule } from './shared/multi-switch';
import { NetqueryModule } from './shared/netquery';
import { NetworkScoutComponent } from './shared/network-scout';
import { NotificationListComponent } from './shared/notification-list/notification-list.component';
import { NotificationComponent } from './shared/notification/notification';
import { CommonPipesModule } from './shared/pipes';
import { ProcessDetailsDialogComponent } from './shared/process-details-dialog';
import { PromptListComponent } from './shared/prompt-list/prompt-list.component';
import { SecurityLockComponent } from './shared/security-lock';
import { SPNAccountDetailsComponent } from './shared/spn-account-details';
import { SPNLoginComponent } from './shared/spn-login';
import { SPNStatusComponent } from './shared/spn-status';
import { PlaceholderComponent } from './shared/text-placeholder';
import { DashboardWidgetComponent } from './pages/dashboard/dashboard-widget/dashboard-widget.component';
import { MergeProfileDialogComponent } from './pages/app-view/merge-profile-dialog/merge-profile-dialog.component';
import { AppInsightsComponent } from './pages/app-view/app-insights/app-insights.component';
import { AppFingerprintPipe } from './pages/app-view/app-fingerprint.pipe';
import { INTEGRATION_SERVICE, integrationServiceFactory } from './integration';
import { SupportProgressDialogComponent } from './pages/support/progress-dialog';
import { I18nModule, I18nService } from './shared/i18n';

function i18nInitializer(i18nService: I18nService) {
  return () => i18nService.initialize();
}

@NgModule({
  declarations: [
    AppComponent,
    NotificationComponent,
    SettingsComponent,
    MonitorPageComponent,
    SideDashComponent,
    NavigationComponent,
    NotificationListComponent,
    PromptListComponent,
    FuzzySearchPipe,
    AppViewComponent,
    QuickSettingInternetButtonComponent,
    QuickSettingUseSPNButtonComponent,
    QuickSettingSelectExitButtonComponent,
    QuickSettingUseSplitTunButtonComponent,
    AppOverviewComponent,
    PlaceholderComponent,
    LoadingComponent,
    ExternalLinkDirective,
    ExitScreenComponent,
    SupportPageComponent,
    SupportFormComponent,
    SecurityLockComponent,
    SPNStatusComponent,
    FeatureScoutComponent,
    SPNLoginComponent,
    SPNAccountDetailsComponent,
    NetworkScoutComponent,
    EditProfileDialog,
    ProcessDetailsDialogComponent,
    QsHistoryComponent,
    DashboardPageComponent,
    DashboardWidgetComponent,
    FeatureCardComponent,
    MergeProfileDialogComponent,
    AppInsightsComponent,
    SupportProgressDialogComponent,
    AppFingerprintPipe
  ],
  imports: [
    BrowserModule,
    CommonModule,
    BrowserAnimationsModule,
    FormsModule,
    ReactiveFormsModule,
    AppRoutingModule,
    FontAwesomeModule,
    OverlayModule,
    PortalModule,
    CdkTableModule,
    DragDropModule,
    MarkdownModule.forRoot(),
    ScrollingModule,
    SfngAccordionModule,
    TabModule,
    SfngTipUpModule.forRoot(MyYamlFile, NotificationsService),
    SfngTooltipModule,
    ActionIndicatorModule,
    SfngDialogModule,
    OverlayStepperModule,
    IntroModule,
    SfngDropDownModule,
    SfngSelectModule,
    SfngMultiSwitchModule,
    SfngMenuModule,
    SfngFocusModule,
    SfngToggleSwitchModule,
    SfngPaginationModule,
    SfngAppIconModule,
    ExpertiseModule,
    ConfigModule,
    I18nModule,
    CountryFlagModule,
    CountIndicatorModule,
    NetqueryModule,
    CommonPipesModule,
    UiModule,
    SPNModule,
    PortmasterAPIModule.forRoot({
      httpAPI: environment.httpAPI,
      websocketAPI: environment.portAPI,
    }),
  ],
  bootstrap: [AppComponent],
  providers: [
    {
      provide: APP_INITIALIZER,
      useFactory: i18nInitializer,
      deps: [I18nService],
      multi: true,
    },
    {
      provide: i18n.NZ_I18N,
      useFactory: (svc: I18nService) => svc.getNzLocale(),
      deps: [I18nService],
    },
    {
      provide: LOCALE_ID,
      useFactory: (svc: I18nService) => svc.getLocaleId(),
      deps: [I18nService],
    },
    {
      provide: INTEGRATION_SERVICE,
      useFactory: integrationServiceFactory
    }
  ]
})
export class AppModule {
  constructor(library: FaIconLibrary) {
    library.addIconPacks(fas, far);
    library.addIcons(faGithub)
  }
}

