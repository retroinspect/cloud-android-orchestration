import { NgIf } from '@angular/common';
import { HttpClientModule } from '@angular/common/http';
import {
  HttpClientTestingModule,
  HttpTestingController,
} from '@angular/common/http/testing';
import {
  ComponentFixture,
  fakeAsync,
  flush,
  TestBed,
} from '@angular/core/testing';
import { ReactiveFormsModule, FormsModule } from '@angular/forms';
import { MatButtonModule } from '@angular/material/button';
import { MatCardModule } from '@angular/material/card';
import { MatCheckboxModule } from '@angular/material/checkbox';
import { MatDividerModule } from '@angular/material/divider';
import { MatExpansionModule } from '@angular/material/expansion';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatIconModule } from '@angular/material/icon';
import { MatInputModule } from '@angular/material/input';
import { MatListModule } from '@angular/material/list';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import { MatSelectModule } from '@angular/material/select';
import { MatSidenavModule } from '@angular/material/sidenav';
import { MatSlideToggleModule } from '@angular/material/slide-toggle';
import { MatSnackBarModule } from '@angular/material/snack-bar';
import { MatToolbarModule } from '@angular/material/toolbar';
import { MatTooltipModule } from '@angular/material/tooltip';
import { BrowserModule } from '@angular/platform-browser';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';
import { RouterModule } from '@angular/router';
import { ActiveEnvPaneComponent } from '../active-env-pane/active-env-pane.component';
import { AppComponent } from '../app.component';
import { CreateHostViewComponent } from '../create-host-view/create-host-view.component';
import { DeviceFormComponent } from '../device-form/device-form.component';
import { EnvCardComponent } from '../env-card/env-card.component';
import { EnvListViewComponent } from '../env-list-view/env-list-view.component';
import { ListRuntimeViewComponent } from '../list-runtime-view/list-runtime-view.component';
import { RegisterRuntimeViewComponent } from '../register-runtime-view/register-runtime-view.component';
import { RuntimeCardComponent } from '../runtime-card/runtime-card.component';
import { SafeUrlPipe } from '../safe-url.pipe';
import { CreateEnvViewComponent } from './create-env-view.component';
import { TestbedHarnessEnvironment } from '@angular/cdk/testing/testbed';
import { HarnessLoader } from '@angular/cdk/testing';
import { MatSelectHarness } from '@angular/material/select/testing';

describe('CreateEnvViewComponent', () => {
  let component: CreateEnvViewComponent;
  let fixture: ComponentFixture<CreateEnvViewComponent>;
  let http: HttpTestingController;
  let loader: HarnessLoader;

  beforeEach(() => {
    TestBed.configureTestingModule({
      declarations: [
        AppComponent,
        EnvListViewComponent,
        ActiveEnvPaneComponent,
        EnvCardComponent,
        CreateEnvViewComponent,
        RegisterRuntimeViewComponent,
        CreateHostViewComponent,
        ListRuntimeViewComponent,
        RuntimeCardComponent,
        DeviceFormComponent,
        SafeUrlPipe,
      ],
      imports: [
        BrowserModule,
        BrowserAnimationsModule,
        HttpClientTestingModule,
        MatButtonModule,
        MatCardModule,
        MatCheckboxModule,
        MatIconModule,
        MatSidenavModule,
        MatSlideToggleModule,
        MatToolbarModule,
        MatTooltipModule,
        MatDividerModule,
        MatInputModule,
        MatFormFieldModule,
        MatSelectModule,
        MatListModule,
        MatExpansionModule,
        MatProgressBarModule,
        NgIf,
        ReactiveFormsModule,
        FormsModule,
        HttpClientModule,
        MatSnackBarModule,
        RouterModule,
      ],
    });
    fixture = TestBed.createComponent(CreateEnvViewComponent);
    component = fixture.componentInstance;
    http = TestBed.inject(HttpTestingController);
    loader = TestbedHarnessEnvironment.loader(fixture);

    fixture.detectChanges();

    let store: any = {
      runtimes: [
        {
          alias: 'runtime1',
          type: 'cloud',
          url: 'http://localhost:8071/api',
          zones: [null, null, null],
          hosts: [],
          status: 'valid',
        },
        {
          alias: 'runtime2',
          url: 'https://sangachoi-dev-dot-cloud-android-orchestrator-dev.googleplex.com',
          hosts: [],
          status: 'error',
        },
      ],
    };

    const mockLocalStorage = {
      getItem: (key: string): string => {
        return JSON.stringify(store[key]);
      },
      setItem: (key: string, value: string) => {
        store[key] = `${value}`;
      },
      removeItem: (key: string) => {
        delete store[key];
      },
      clear: () => {
        store = {};
      },
    };


    spyOn(window.localStorage, 'getItem').and.callFake(
      mockLocalStorage.getItem
    );
    spyOn(window.localStorage, 'setItem').and.callFake(
      mockLocalStorage.setItem
    );
    spyOn(window.localStorage, 'removeItem').and.callFake(
      mockLocalStorage.removeItem
    );
    spyOn(window.localStorage, 'clear').and.callFake(mockLocalStorage.clear);

    http.expectOne('http://localhost:8071/api/info').flush({
      type: 'cloud',
    });

    http.expectOne('http://localhost:8071/api/v1/zones').flush({});
  });

  it('localStorage set up', fakeAsync(() => {
    flush();

    expect(window.localStorage.getItem('runtimes')).toBeDefined();
    expect(JSON.parse(window.localStorage.getItem('runtimes')!)?.length).toBe(
      2
    );
  }));

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('press runtime selector', fakeAsync(async () => {
    flush();
    const selectors = await loader.getAllHarnesses(MatSelectHarness);
    console.log(await selectors[0].getValueText());
    await selectors[0].open();
    const options = await selectors[0].getOptions();
    console.log(options);
    console.log('?????');
    expect(options.length).toBe(2);
    expect(await options[1].getText()).toBe('runtime1');
  }));
});
