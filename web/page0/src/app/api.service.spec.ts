import { TestBed } from '@angular/core/testing';
import {
  HttpClientTestingModule,
  HttpTestingController,
} from '@angular/common/http/testing';

import { ApiService } from './api.service';

describe('ApiService', () => {
  let service: ApiService;
  let http: HttpTestingController;
  const testRuntime = 'http://test-runtime.com';

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [HttpClientTestingModule],
      providers: [],
    });

    service = TestBed.inject(ApiService);
    http = TestBed.inject(HttpTestingController);
  });

  it('should be created', () => {
    expect(service).toBeTruthy();
  });

  it('should be able to request list hosts', async () => {
    service.listHosts(testRuntime, 'us-east1-a').subscribe((v) => {
      expect(v).toEqual({});
    });

    http.expectOne(`${testRuntime}/v1/zones/us-east1-a/hosts`).flush({});
  });
});
