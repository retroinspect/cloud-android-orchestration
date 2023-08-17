import { HttpClientTestingModule } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';

import { DeviceFormService } from './device-form.service';

describe('DeviceFormService', () => {
  let service: DeviceFormService;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [HttpClientTestingModule],
      providers: [],
    });

    service = TestBed.inject(DeviceFormService);
  });

  it('should be created', () => {
    expect(service).toBeTruthy();
  });
});
